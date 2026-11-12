// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipelinehistory"
	"github.com/pocketbase/pocketbase/core"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

// ReadActionAuthorPipelineRunActivityName reads the failed wallet action
// steps of a pipeline run for the agent.
const ReadActionAuthorPipelineRunActivityName = "Read action author pipeline run"

const executionStatusPrefix = "WORKFLOW_EXECUTION_STATUS_"

// ReadActionAuthorPipelineRunInput is the run URL, run ID or pipeline the
// user gave the agent.
type ReadActionAuthorPipelineRunInput struct {
	Text string `json:"text"`
}

// ReadActionAuthorPipelineRunOutput is a pipeline run, its failed
// mobile-automation steps in pipeline order and every other failed step as
// "<step_id> (<use>): <message>". Its lists are never null.
type ReadActionAuthorPipelineRunOutput struct {
	Pipeline      string                            `json:"pipeline"`
	WorkflowID    string                            `json:"workflow_id"`
	RunID         string                            `json:"run_id"`
	Status        string                            `json:"status"`
	Failures      []actionauthor.PipelineRunFailure `json:"failures"`
	OtherFailures []string                          `json:"other_failures"`
}

// ReadActionAuthorPipelineRunActivity reads a failed pipeline run of the
// caller session's namespace.
type ReadActionAuthorPipelineRunActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
}

// NewReadActionAuthorPipelineRunActivity binds the run reader to the application.
func NewReadActionAuthorPipelineRunActivity(app core.App) *ReadActionAuthorPipelineRunActivity {
	return &ReadActionAuthorPipelineRunActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: ReadActionAuthorPipelineRunActivityName,
		},
		authorSessionLookup: newAuthorSessionLookup(app),
		app:                 app,
	}
}

// Name returns the registered Temporal activity name.
func (a *ReadActionAuthorPipelineRunActivity) Name() string { return a.BaseActivity.Name }

// Execute reads the run for the activity's actual caller.
func (a *ReadActionAuthorPipelineRunActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[ReadActionAuthorPipelineRunInput](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	output, err := a.read(ctx, activityCaller(ctx), payload)
	if err != nil {
		return result, err
	}
	result.Output = output
	return result, nil
}

// read accepts only the ActionAuthor child of a running session and reads
// the referenced run, or the latest failed run of the referenced pipeline,
// in the namespace of the session's organization.
func (a *ReadActionAuthorPipelineRunActivity) read(
	ctx context.Context,
	caller authorCaller,
	payload ReadActionAuthorPipelineRunInput,
) (ReadActionAuthorPipelineRunOutput, error) {
	var output ReadActionAuthorPipelineRunOutput
	selection, err := a.sessionSelection(ctx, &a.BaseActivity, a.app, caller)
	if err != nil {
		return output, err
	}
	ref, err := actionauthor.ParsePipelineRunReference(payload.Text)
	if err != nil {
		return output, actionAuthorError(&a.BaseActivity, err)
	}
	// Pipelines run in the session organization's namespace, not in the
	// runner namespace the session runs in.
	c, err := a.temporalClient(selection.Organization.GetString("canonified_name"))
	if err != nil {
		return output, lookupError(&a.BaseActivity, fmt.Errorf("read pipeline run: %w", err))
	}
	if ref.Pipeline != "" {
		ref, err = a.latestFailedRun(ctx, c, ref.Pipeline, selection.Organization.Id)
		if err != nil {
			return output, err
		}
	}
	described, err := c.DescribeWorkflowExecution(ctx, ref.WorkflowID, ref.RunID)
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return output, actionAuthorError(&a.BaseActivity, notAPipelineRun(ref.WorkflowID))
	}
	if err != nil {
		return output, lookupError(&a.BaseActivity, fmt.Errorf("read pipeline run: %w", err))
	}
	info := described.GetWorkflowExecutionInfo()
	if info.GetType().GetName() != pipelineWorkflowName {
		return output, actionAuthorError(&a.BaseActivity, notAPipelineRun(ref.WorkflowID))
	}
	output = ReadActionAuthorPipelineRunOutput{
		Pipeline:   pipelineIdentifier(info.GetSearchAttributes().GetIndexedFields()),
		WorkflowID: info.GetExecution().GetWorkflowId(),
		RunID:      info.GetExecution().GetRunId(),
		Status: strings.TrimPrefix(
			enumspb.WorkflowExecutionStatus_name[int32(info.GetStatus())],
			executionStatusPrefix,
		),
		Failures:      []actionauthor.PipelineRunFailure{},
		OtherFailures: []string{},
	}
	switch info.GetStatus() {
	case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED:
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return output, nil
	default:
		return ReadActionAuthorPipelineRunOutput{}, actionAuthorError(&a.BaseActivity, fmt.Errorf(
			"%w: run %s did not fail (status %s)",
			actionauthor.ErrInvalid,
			ref.WorkflowID,
			output.Status,
		))
	}
	if err := a.readFailures(ctx, c, selection, &output); err != nil {
		return ReadActionAuthorPipelineRunOutput{}, err
	}
	return output, nil
}

// latestFailedRun finds the newest failed run in the caller's namespace of
// a pipeline the organization owns or that is published.
func (a *ReadActionAuthorPipelineRunActivity) latestFailedRun(
	ctx context.Context,
	c client.Client,
	path, orgID string,
) (actionauthor.PipelineRunRef, error) {
	_, err := actionauthor.CheckRecordAccess(a.app, "pipelines", path, orgID)
	if errors.Is(err, actionauthor.ErrInvalid) {
		return actionauthor.PipelineRunRef{}, actionAuthorError(&a.BaseActivity, fmt.Errorf(
			"%w: pipeline %s not found", actionauthor.ErrInvalid, path,
		))
	}
	if err != nil {
		return actionauthor.PipelineRunRef{}, actionAuthorError(&a.BaseActivity, err)
	}
	identifier := workflowengine.NormalizePipelineIdentifier(path)
	listed, err := c.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		PageSize: 1,
		Query:    latestFailedRunQuery(identifier),
	})
	if err != nil {
		return actionauthor.PipelineRunRef{}, lookupError(
			&a.BaseActivity,
			fmt.Errorf("list failed runs of %s: %w", identifier, err),
		)
	}
	if len(listed.GetExecutions()) == 0 {
		return actionauthor.PipelineRunRef{}, actionAuthorError(&a.BaseActivity, fmt.Errorf(
			"%w: no failed run of pipeline %s in your organization",
			actionauthor.ErrInvalid,
			identifier,
		))
	}
	execution := listed.GetExecutions()[0].GetExecution()
	return actionauthor.PipelineRunRef{
		WorkflowID: execution.GetWorkflowId(),
		RunID:      execution.GetRunId(),
	}, nil
}

// latestFailedRunQuery lists the failed runs of a pipeline, newest first.
func latestFailedRunQuery(identifier string) string {
	return fmt.Sprintf(
		"WorkflowType = '%s' AND %s = '%s' AND ExecutionStatus = 'Failed'",
		pipelineWorkflowName,
		workflowengine.PipelineIdentifierSearchAttribute,
		strings.ReplaceAll(identifier, "'", `\'`),
	)
}

func notAPipelineRun(workflowID string) error {
	return fmt.Errorf("%w: run %s is not a pipeline run", actionauthor.ErrInvalid, workflowID)
}

// pipelineIdentifier reads the PipelineIdentifier search attribute.
func pipelineIdentifier(fields map[string]*commonpb.Payload) string {
	payload, ok := fields[workflowengine.PipelineIdentifierSearchAttribute]
	if !ok {
		return ""
	}
	var identifier string
	if converter.GetDefaultDataConverter().FromPayload(payload, &identifier) != nil {
		return ""
	}
	return workflowengine.NormalizePipelineIdentifier(identifier)
}

// pipelineFailureDetails is the details of a pipeline's close failure:
// one WorkflowError per failed step, carrying its details.step_id.
type pipelineFailureDetails struct {
	Errors []workflowengine.WorkflowError `json:"errors"`
}

// readFailures reads the failed run's close failure and sorts the failed
// steps, in pipeline order, into wallet action failures and other failures.
func (a *ReadActionAuthorPipelineRunActivity) readFailures(
	ctx context.Context,
	c client.Client,
	selection actionauthor.Selection,
	output *ReadActionAuthorPipelineRunOutput,
) error {
	failure, err := closeFailure(ctx, c, output.WorkflowID, output.RunID)
	if err != nil {
		return lookupError(&a.BaseActivity, err)
	}
	// A failure without step errors, such as a definition that did not
	// parse, has only its own message.
	details, decodeErr := workflowengine.DecodePayload[pipelineFailureDetails](failure.Details)
	if decodeErr != nil {
		details.Errors = nil
	}
	if len(details.Errors) == 0 {
		output.OtherFailures = append(output.OtherFailures, failureMessage(failure))
		return nil
	}
	run, err := pipelinehistory.Load(ctx, c, output.WorkflowID, output.RunID, unknownOutputKind)
	if err != nil {
		return lookupError(&a.BaseActivity, err)
	}
	byStep := make(map[string]workflowengine.WorkflowError, len(details.Errors))
	for _, stepError := range details.Errors {
		stepID, _ := stepError.Details["step_id"].(string)
		if _, seen := byStep[stepID]; !seen {
			byStep[stepID] = stepError
		}
	}
	for i, step := range run.Definition.Steps {
		stepError, failed := byStep[step.ID]
		if !failed {
			continue
		}
		delete(byStep, step.ID)
		if step.Use != actionauthor.StepUseMobileAutomation ||
			stepError.Code == errorcodes.PipelineInputError {
			output.OtherFailures = append(output.OtherFailures,
				fmt.Sprintf("%s (%s): %s", step.ID, step.Use, failureMessage(stepError)))
			continue
		}
		stdout, _ := stepError.Details["stdout"].(string)
		described, err := actionauthor.DescribeFailedStep(
			a.app,
			selection,
			run.Definition,
			i,
			actionauthor.StepFailure{
				Code:    stepError.Code,
				Message: failureMessage(stepError),
				Stdout:  stdout,
			},
		)
		if err != nil {
			return actionAuthorError(&a.BaseActivity, err)
		}
		output.Failures = append(output.Failures, described)
	}
	for _, stepError := range details.Errors {
		stepID, _ := stepError.Details["step_id"].(string)
		if _, unmatched := byStep[stepID]; unmatched {
			delete(byStep, stepID)
			output.OtherFailures = append(output.OtherFailures,
				fmt.Sprintf("%s (unknown step): %s", stepID, failureMessage(stepError)))
		}
	}
	return nil
}

// closeFailure reads the failure of a failed run's close event.
func closeFailure(
	ctx context.Context,
	c client.Client,
	workflowID, runID string,
) (workflowengine.WorkflowError, error) {
	iter := c.GetWorkflowHistory(
		ctx,
		workflowID,
		runID,
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
	)
	if !iter.HasNext() {
		return workflowengine.WorkflowError{}, fmt.Errorf("run %s has no close event", workflowID)
	}
	event, err := iter.Next()
	if err != nil {
		return workflowengine.WorkflowError{}, fmt.Errorf(
			"read close event of %s: %w",
			workflowID,
			err,
		)
	}
	return workflowengine.ParseWorkflowFailure(
		event.GetWorkflowExecutionFailedEventAttributes().GetFailure(),
	), nil
}

// failureMessage is a step failure's message, or its summary without one.
func failureMessage(failure workflowengine.WorkflowError) string {
	if message := strings.TrimSpace(failure.Message); message != "" {
		return message
	}
	return strings.TrimSpace(failure.Summary)
}

// unknownOutputKind leaves step outputs to pipelinehistory's default: only
// the run's definition is read.
func unknownOutputKind(string) (workflowengine.OutputKind, bool) {
	return workflowengine.OutputMap, false
}
