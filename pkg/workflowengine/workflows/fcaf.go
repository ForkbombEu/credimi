// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"gopkg.in/yaml.v3"
)

const FCAFAssessmentTaskQueue = "FCAFAssessmentTaskQueue"
const fcafPipelineTaskQueue = "PipelineTaskQueue"

type FCAFAssessmentWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type FCAFTestWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type FCAFSingleTestWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type FCAFAssessmentWorkflowPayload struct {
	TestIDs     []string        `json:"test_ids"               yaml:"test_ids"`
	Suite       string          `json:"suite,omitempty"        yaml:"suite,omitempty"`
	CatalogRoot string          `json:"catalog_root,omitempty" yaml:"catalog_root,omitempty"`
	Evidence    evidence.Bundle `json:"evidence,omitempty"     yaml:"evidence,omitempty"`
	Runtime     map[string]any  `json:"runtime,omitempty"      yaml:"runtime,omitempty"`
	RunnerID    string          `json:"runner_id,omitempty"    yaml:"runner_id,omitempty"`
}

type FCAFPreconditionPipelineWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type FCAFPreconditionPipelineWorkflowPayload struct {
	Precondition dsl.PreconditionDefinition `json:"precondition"`
	Runtime      map[string]any             `json:"runtime,omitempty"`
}

type FCAFPipelineResultReference struct {
	Aliases    []string `json:"aliases"`
	WorkflowID string   `json:"workflow_id"`
	RunID      string   `json:"run_id,omitempty"`
	Namespace  string   `json:"namespace"`
}

type FCAFTestWorkflowPayload struct {
	TestID                    string                        `json:"test_id,omitempty"`
	TestIDs                   []string                      `json:"test_ids,omitempty"`
	Suite                     string                        `json:"suite"`
	CatalogRoot               string                        `json:"catalog_root,omitempty"`
	Runtime                   map[string]any                `json:"runtime,omitempty"`
	PipelineReferences        []FCAFPipelineResultReference `json:"pipeline_references,omitempty"`
	TestPipelinePreconditions map[string][]string           `json:"test_pipeline_preconditions,omitempty"`
}

type FCAFSingleTestWorkflowPayload struct {
	TestID             string                        `json:"test_id"`
	Suite              string                        `json:"suite"`
	CatalogRoot        string                        `json:"catalog_root,omitempty"`
	Runtime            map[string]any                `json:"runtime,omitempty"`
	PipelineReferences []FCAFPipelineResultReference `json:"pipeline_references,omitempty"`
}

type fcafPipelineWorkflowInput struct {
	WorkflowDefinition *pipelineinternal.WorkflowDefinition `yaml:"workflow_definition"       json:"workflow_definition"`
	WorkflowInput      workflowengine.WorkflowInput         `yaml:"workflow_input"            json:"workflow_input"`
	Debug              bool                                 `yaml:"debug,omitempty"           json:"debug,omitempty"`
	ParentRunData      map[string]any                       `yaml:"parent_run_data,omitempty" json:"parent_run_data,omitempty"`
}

const fcafPreconditionPollInterval = 5 * time.Second

var fcafStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

func NewFCAFAssessmentWorkflow() *FCAFAssessmentWorkflow {
	w := &FCAFAssessmentWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func NewFCAFPreconditionPipelineWorkflow() *FCAFPreconditionPipelineWorkflow {
	w := &FCAFPreconditionPipelineWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func NewFCAFTestWorkflow() *FCAFTestWorkflow {
	w := &FCAFTestWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func NewFCAFSingleTestWorkflow() *FCAFSingleTestWorkflow {
	w := &FCAFSingleTestWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (w *FCAFAssessmentWorkflow) Name() string {
	return "Run FCAF assessment"
}

func (w *FCAFPreconditionPipelineWorkflow) Name() string {
	return "Run FCAF precondition pipeline"
}

func (w *FCAFTestWorkflow) Name() string {
	return "Run FCAF tests"
}

func (w *FCAFSingleTestWorkflow) Name() string {
	return "Run FCAF test"
}

func (w *FCAFAssessmentWorkflow) GetOptions() workflow.ActivityOptions {
	ao := DefaultActivityOptions
	ao.RetryPolicy.MaximumAttempts = 1
	return ao
}

func (w *FCAFPreconditionPipelineWorkflow) GetOptions() workflow.ActivityOptions {
	ao := DefaultActivityOptions
	ao.RetryPolicy.MaximumAttempts = 1
	return ao
}

func (w *FCAFTestWorkflow) GetOptions() workflow.ActivityOptions {
	ao := DefaultActivityOptions
	ao.RetryPolicy.MaximumAttempts = 1
	return ao
}

func (w *FCAFSingleTestWorkflow) GetOptions() workflow.ActivityOptions {
	ao := DefaultActivityOptions
	ao.RetryPolicy.MaximumAttempts = 1
	return ao
}

func (w *FCAFAssessmentWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *FCAFPreconditionPipelineWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *FCAFTestWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *FCAFSingleTestWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *FCAFAssessmentWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	payload, err := workflowengine.DecodePayload[FCAFAssessmentWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}
	payload.Runtime = cloneAnyMap(payload.Runtime)
	if payload.Runtime == nil {
		payload.Runtime = map[string]any{}
	}
	if runnerID := strings.TrimSpace(payload.RunnerID); runnerID != "" {
		if existing, ok := payload.Runtime["runner_id"].(string); ok &&
			strings.TrimSpace(existing) != "" &&
			existing != runnerID {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
				fmt.Errorf("runner_id conflicts with runtime.runner_id"),
				input.RunMetadata,
			)
		}
		payload.Runtime["runner_id"] = runnerID
	}

	resolveAct := activities.NewFCAFResolveExecutionPlanActivity()
	var resolveResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, resolveAct.Name(), workflowengine.ActivityInput{
		Payload: activities.FCAFResolveExecutionPlanActivityInput{
			TestIDs:     payload.TestIDs,
			Suite:       payload.Suite,
			CatalogRoot: payload.CatalogRoot,
			Runtime:     payload.Runtime,
		},
	}).Get(ctx, &resolveResult); err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}

	executionPlan, err := decodeFCAFExecutionPlan(resolveResult.Output)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(workflowengine.WorkflowError{
				Code:    errorcodes.Codes[errorcodes.UnexpectedActivityOutput].Code,
				Summary: errorcodes.Codes[errorcodes.UnexpectedActivityOutput].Description,
				Message: err.Error(),
			}),
			input.RunMetadata,
		)
	}

	payload.Evidence.Runtime = payload.Runtime
	if payload.Evidence.PipelineOutputs == nil {
		payload.Evidence.PipelineOutputs = map[string]any{}
	}
	if appURL := lookupStringMap(input.Config, "app_url"); appURL != "" {
		payload.Runtime["app_url"] = appURL
	}
	pipelineReferences := make(
		[]FCAFPipelineResultReference,
		0,
		len(executionPlan.PipelinePreconditions),
	)
	for _, precondition := range executionPlan.PipelinePreconditions {
		if sharedOutput, ok := existingFCAFPipelineOutput(
			payload.Evidence.PipelineOutputs,
			precondition,
		); ok {
			payload.Evidence.PipelineOutputs[precondition.ID] = sharedOutput
			pipelineResult, decodeErr := evidence.DecodePipelineExecutionResult(sharedOutput)
			if decodeErr != nil {
				return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
					decodeErr,
					input.RunMetadata,
				)
			}
			pipelineReferences = addFCAFPipelineReference(
				pipelineReferences,
				precondition,
				pipelineResult,
				workflow.GetInfo(ctx).Namespace,
			)
			continue
		}
		pipelineID := strings.TrimSpace(precondition.PipelineID)
		childOutput, err := w.runPreconditionPipeline(ctx, input, precondition, payload.Runtime)
		if err != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				input.RunMetadata,
			)
		}
		payload.Evidence.PipelineOutputs[precondition.ID] = childOutput
		if pipelineID != "" {
			payload.Evidence.PipelineOutputs[pipelineID] = childOutput
		}
		pipelineReferences = addFCAFPipelineReference(
			pipelineReferences,
			precondition,
			childOutput,
			workflow.GetInfo(ctx).Namespace,
		)
	}

	report := engine.Report{
		Suite:           payload.Suite,
		SelectedTestIDs: append([]string(nil), executionPlan.SelectedTests...),
		Evidence:        engine.EvidenceMap{},
	}
	if len(executionPlan.SelectedTests) > 0 {
		childReport, childErr := w.runTestsChild(ctx, input, FCAFTestWorkflowPayload{
			TestIDs:                   append([]string(nil), executionPlan.SelectedTests...),
			Suite:                     payload.Suite,
			CatalogRoot:               payload.CatalogRoot,
			Runtime:                   cloneAnyMap(payload.Runtime),
			PipelineReferences:        pipelineReferences,
			TestPipelinePreconditions: executionPlan.TestPipelinePreconditions,
		})
		if childErr != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				childErr,
				input.RunMetadata,
			)
		}
		report = childReport
	}
	report.Suite = payload.Suite
	report.SelectedTestIDs = append([]string(nil), executionPlan.SelectedTests...)
	report.Status = fcafAggregateStatus(report.ExecutedTests)
	report.Summary = fcafSummaryFromExecutedTests(report.ExecutedTests)
	publicReport := report.PublicReport()
	if publicReport.Status != "passed" {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: "FCAF assessment failed",
				Message: summarizeFCAFFailures(report),
				Details: map[string]any{
					"output": publicReport,
				},
			}),
			input.RunMetadata,
		)
	}

	return workflowengine.WorkflowResult{
		Message: "FCAF assessment completed",
		Output:  publicReport,
	}, nil
}

func (w *FCAFTestWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	payload, err := workflowengine.DecodePayload[FCAFTestWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}
	testIDs := normalizedFCAFTestIDs(payload)
	if len(testIDs) == 0 {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			fmt.Errorf("test_ids is required"),
			input.RunMetadata,
		)
	}

	report := engine.Report{
		Suite:           payload.Suite,
		SelectedTestIDs: append([]string(nil), testIDs...),
		Evidence:        engine.EvidenceMap{},
	}
	for _, testID := range testIDs {
		childReport, childErr := w.runSingleTestChild(ctx, input, FCAFSingleTestWorkflowPayload{
			TestID:      testID,
			Suite:       payload.Suite,
			CatalogRoot: payload.CatalogRoot,
			Runtime:     cloneAnyMap(payload.Runtime),
			PipelineReferences: filterFCAFTestPipelineReferences(
				payload.PipelineReferences,
				payload.TestPipelinePreconditions,
				payload.TestPipelinePreconditions[testID],
			),
		})
		if childErr != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				childErr,
				input.RunMetadata,
			)
		}
		mergeFCAFPublicReport(&report, childReport)
	}
	report.Status = fcafAggregateStatus(report.ExecutedTests)
	report.Summary = fcafSummaryFromExecutedTests(report.ExecutedTests)

	return workflowengine.WorkflowResult{
		Message: "FCAF tests completed",
		Output:  report.PublicReport(),
	}, nil
}

func (w *FCAFSingleTestWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	payload, err := workflowengine.DecodePayload[FCAFSingleTestWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}
	testID := strings.TrimSpace(payload.TestID)
	if testID == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			fmt.Errorf("test_id is required"),
			input.RunMetadata,
		)
	}

	bundle, err := fcafPipelineEvidenceBundle(ctx, payload.Runtime, payload.PipelineReferences)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}

	assessmentActivity := activities.NewFCAFAssessmentActivity()
	var assessmentResult workflowengine.ActivityResult
	err = workflow.ExecuteActivity(ctx, assessmentActivity.Name(), workflowengine.ActivityInput{
		Payload: activities.FCAFAssessmentActivityInput{
			TestIDs:     []string{testID},
			Suite:       payload.Suite,
			CatalogRoot: payload.CatalogRoot,
			Evidence:    bundle,
			Runtime:     payload.Runtime,
		},
	}).Get(ctx, &assessmentResult)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}
	output, decodeErr := decodeFCAFAssessmentActivityOutput(assessmentResult.Output)
	if decodeErr != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			decodeErr,
			input.RunMetadata,
		)
	}
	if output.Report.Status == "" {
		output.Report.PopulateDerivedViews()
	}
	keepOnlyFCAFExecutedTests(&output.Report, []string{testID})
	stripFCAFEvidenceValues(&output.Report)

	return workflowengine.WorkflowResult{
		Message: "FCAF test completed",
		Output:  output.Report.PublicReport(),
	}, nil
}

func fcafPipelineEvidenceBundle(
	ctx workflow.Context,
	runtime map[string]any,
	references []FCAFPipelineResultReference,
) (evidence.Bundle, error) {
	bundle := evidence.Bundle{
		PipelineOutputs: map[string]any{},
		Runtime:         cloneAnyMap(runtime),
	}
	if bundle.Runtime == nil {
		bundle.Runtime = map[string]any{}
	}
	resultActivity := activities.NewGetWorkflowResultActivity()
	retrieved := map[string]evidence.PipelineExecutionResult{}
	for _, reference := range references {
		if _, exists := bundle.Runtime["namespace"]; !exists {
			bundle.Runtime["namespace"] = reference.Namespace
		}
		cacheKey := reference.Namespace + "\x00" + reference.WorkflowID + "\x00" + reference.RunID
		pipelineResult, ok := retrieved[cacheKey]
		if !ok {
			var activityResult workflowengine.ActivityResult
			err := workflow.ExecuteActivity(ctx, resultActivity.Name(), workflowengine.ActivityInput{
				Payload: activities.GetWorkflowResultActivityInput{
					WorkflowID:           reference.WorkflowID,
					RunID:                reference.RunID,
					WorkflowNamespace:    reference.Namespace,
					ReturnFailureDetails: true,
				},
			}).
				Get(ctx, &activityResult)
			if err != nil {
				return evidence.Bundle{}, err
			}
			workflowResult, decodeErr := decodeWorkflowResult(activityResult.Output)
			if decodeErr != nil {
				return evidence.Bundle{}, decodeErr
			}
			pipelineResult, decodeErr = fcafPipelineExecutionResult(
				workflowResult,
				reference.WorkflowID,
				reference.RunID,
			)
			if decodeErr != nil {
				return evidence.Bundle{}, decodeErr
			}
			retrieved[cacheKey] = pipelineResult
		}
		for _, alias := range reference.Aliases {
			bundle.PipelineOutputs[alias] = pipelineResult
		}
	}
	return bundle, nil
}

func (w *FCAFTestWorkflow) runSingleTestChild(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	payload FCAFSingleTestWorkflowPayload,
) (engine.Report, error) {
	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: fcafSingleTestWorkflowID(
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			payload.TestID,
		),
		TaskQueue:         FCAFAssessmentTaskQueue,
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
	})
	var result workflowengine.WorkflowResult
	err := workflow.ExecuteChildWorkflow(
		childCtx,
		NewFCAFSingleTestWorkflow().Name(),
		workflowengine.WorkflowInput{
			Payload: payload,
			Config:  cloneAnyMap(input.Config),
		},
	).Get(childCtx, &result)
	if err != nil {
		return engine.Report{}, err
	}
	var report engine.Report
	data, err := json.Marshal(result.Output)
	if err != nil {
		return report, fmt.Errorf("marshal FCAF single test output: %w", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return report, fmt.Errorf("decode FCAF single test output: %w", err)
	}
	return report, nil
}

func (w *FCAFAssessmentWorkflow) runTestsChild(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	payload FCAFTestWorkflowPayload,
) (engine.Report, error) {
	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:        fcafTestsWorkflowID(workflow.GetInfo(ctx).WorkflowExecution.ID),
		TaskQueue:         FCAFAssessmentTaskQueue,
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
	})
	var result workflowengine.WorkflowResult
	err := workflow.ExecuteChildWorkflow(
		childCtx,
		NewFCAFTestWorkflow().Name(),
		workflowengine.WorkflowInput{
			Payload: payload,
			Config:  cloneAnyMap(input.Config),
		},
	).Get(childCtx, &result)
	if err != nil {
		return engine.Report{}, err
	}
	var report engine.Report
	data, err := json.Marshal(result.Output)
	if err != nil {
		return report, fmt.Errorf("marshal FCAF test output: %w", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return report, fmt.Errorf("decode FCAF test output: %w", err)
	}
	return report, nil
}

func existingFCAFPipelineOutput(
	outputs map[string]any,
	precondition dsl.PreconditionDefinition,
) (any, bool) {
	if outputs == nil {
		return nil, false
	}
	if output, ok := outputs[precondition.ID]; ok {
		return output, true
	}
	pipelineID := strings.TrimSpace(precondition.PipelineID)
	if pipelineID == "" {
		return nil, false
	}
	output, ok := outputs[pipelineID]
	return output, ok
}

func (w *FCAFAssessmentWorkflow) runPreconditionPipeline(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	precondition dsl.PreconditionDefinition,
	runtime map[string]any,
) (evidence.PipelineExecutionResult, error) {
	childOpts := workflow.ChildWorkflowOptions{
		WorkflowID: fcafPreconditionWorkflowID(
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			precondition.ID,
		),
		TaskQueue:         FCAFAssessmentTaskQueue,
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
	}
	childCtx := workflow.WithChildOptions(ctx, childOpts)

	var result workflowengine.WorkflowResult
	err := workflow.ExecuteChildWorkflow(
		childCtx,
		NewFCAFPreconditionPipelineWorkflow().Name(),
		workflowengine.WorkflowInput{
			Payload: FCAFPreconditionPipelineWorkflowPayload{
				Precondition: precondition,
				Runtime:      cloneAnyMap(runtime),
			},
			Config: cloneAnyMap(input.Config),
		},
	).Get(childCtx, &result)
	if err != nil {
		return evidence.PipelineExecutionResult{}, err
	}
	return evidence.DecodePipelineExecutionResult(result.Output)
}

func (w *FCAFPreconditionPipelineWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	payload, err := workflowengine.DecodePayload[FCAFPreconditionPipelineWorkflowPayload](
		input.Payload,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	appURL, _ := input.Config["app_url"].(string)
	if strings.TrimSpace(appURL) == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}
	input.Config = cloneAnyMap(input.Config)
	input.Config[workflowengine.CollectPipelineStepFailuresConfigKey] = true
	pipelineID := strings.TrimSpace(payload.Precondition.PipelineID)
	if pipelineID == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			fmt.Errorf("pipeline_id is required"),
			input.RunMetadata,
		)
	}
	namespace, err := pipelineNamespaceFromID(pipelineID)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	pipelineYAML, err := fetchFCAFPreconditionPipelineYAML(ctx, appURL, pipelineID)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}
	rewrittenYAML, runnerInfo, err := prepareFCAFPreconditionPipelineYAML(
		pipelineYAML,
		payload.Runtime,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(workflowengine.WorkflowError{
				Code:    errorcodes.Codes[errorcodes.PipelineParsingError].Code,
				Summary: errorcodes.Codes[errorcodes.PipelineParsingError].Description,
				Message: err.Error(),
			}),
			input.RunMetadata,
		)
	}

	runnerIDs := fcafRunnerIDsWithGlobal(runnerInfo, lookupStringMap(payload.Runtime, "runner_id"))
	if len(runnerIDs) == 0 {
		return w.runDirectPipelineChild(ctx, input, rewrittenYAML)
	}
	return w.runQueuedPipeline(ctx, input, pipelineID, namespace, rewrittenYAML, runnerIDs)
}

func (w *FCAFPreconditionPipelineWorkflow) runDirectPipelineChild(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	pipelineYAML string,
) (workflowengine.WorkflowResult, error) {
	wfDef, err := pipelineinternal.ParseWorkflow(pipelineYAML)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	childOpts := workflow.ChildWorkflowOptions{
		WorkflowID: fmt.Sprintf("fcaf-pipeline-%s", uuid.NewString()),
		TaskQueue:  fcafPipelineTaskQueue,
	}
	childCtx := workflow.WithChildOptions(ctx, childOpts)
	var childResult workflowengine.WorkflowResult
	err = workflow.ExecuteChildWorkflow(
		childCtx,
		"Dynamic Pipeline Workflow",
		fcafPipelineWorkflowInput{
			WorkflowDefinition: wfDef,
			WorkflowInput: workflowengine.WorkflowInput{
				Config: cloneAnyMap(input.Config),
			},
			Debug: wfDef.Runtime.Debug,
		},
	).Get(childCtx, &childResult)
	if err != nil {
		failedResult, recovered := failedPipelineWorkflowResult(err)
		if !recovered {
			return workflowengine.WorkflowResult{}, err
		}
		childResult = failedResult
	}
	pipelineResult, err := fcafPipelineExecutionResult(
		childResult,
		childResult.WorkflowID,
		childResult.WorkflowRunID,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	return workflowengine.WorkflowResult{
		Message: "FCAF precondition pipeline completed",
		Output:  pipelineResult,
	}, nil
}

func (w *FCAFPreconditionPipelineWorkflow) runQueuedPipeline(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	pipelineID string,
	namespace string,
	pipelineYAML string,
	runnerIDs []string,
) (workflowengine.WorkflowResult, error) {
	config := cloneAnyMap(input.Config)
	if config == nil {
		config = map[string]any{}
	}
	if runnerID := lookupStringMap(config, "global_runner_id"); runnerID == "" {
		if runnerID = lookupStringMap(input.Config, "runner_id"); runnerID == "" {
			runnerID = lookupStringMap(config, "runner_id")
		}
		if runnerID == "" {
			runnerID = runnerIDs[0]
		}
		config["global_runner_id"] = runnerID
	}
	config["namespace"] = namespace

	enqueueAct := activities.NewEnqueuePipelineRunTicketActivity()
	ticketID := fcafQueueTicketID(workflow.GetInfo(ctx).WorkflowExecution.ID)
	var enqueueResult workflowengine.ActivityResult
	err := workflow.ExecuteActivity(ctx, enqueueAct.Name(), workflowengine.ActivityInput{
		Payload: activities.EnqueuePipelineRunTicketActivityInput{
			TicketID:           ticketID,
			OwnerNamespace:     namespace,
			EnqueuedAt:         workflow.Now(ctx).UTC(),
			RunnerIDs:          runnerIDs,
			PipelineIdentifier: pipelineID,
			YAML:               pipelineYAML,
			PipelineConfig:     config,
			Memo:               map[string]any{"test": "fcaf-precondition"},
		},
	}).Get(ctx, &enqueueResult)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	enqueuedStatus, err := decodeEnqueuePipelineRunTicketOutput(enqueueResult.Output)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	queryAct := activities.NewQueryMobileRunnerSemaphoreRunStatusActivity()
	checkAct := activities.NewCheckWorkflowClosedActivity()
	getResultAct := activities.NewGetWorkflowResultActivity()
	lastKnownStatus := enqueueRunTicketStatusToSemaphoreStatus(ticketID, runnerIDs, enqueuedStatus)

	for {
		status := lastKnownStatus
		if status.Status == "" ||
			status.Status == mobilerunnersemaphore.MobileRunnerSemaphoreRunQueued ||
			status.Status == mobilerunnersemaphore.MobileRunnerSemaphoreRunStarting {
			var statusResult workflowengine.ActivityResult
			if err := workflow.ExecuteActivity(ctx, queryAct.Name(), workflowengine.ActivityInput{
				Payload: activities.QueryMobileRunnerSemaphoreRunStatusInput{
					RunnerID:       runnerIDs[0],
					OwnerNamespace: namespace,
					TicketID:       ticketID,
				},
			}).Get(ctx, &statusResult); err != nil {
				return workflowengine.WorkflowResult{}, err
			}

			status, err = decodeSemaphoreRunStatus(statusResult.Output)
			if err != nil {
				return workflowengine.WorkflowResult{}, err
			}
			status = mergeSemaphoreStatus(lastKnownStatus, status)
			lastKnownStatus = status
		}
		if isTerminalSemaphoreStatus(status.Status) &&
			(strings.TrimSpace(status.WorkflowID) == "" || strings.TrimSpace(status.WorkflowNamespace) == "") {
			return workflowengine.WorkflowResult{}, workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
					Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
					Message: firstNonEmpty(
						status.ErrorMessage,
						"fcaf precondition pipeline did not start",
					),
				},
			)
		}

		if strings.TrimSpace(status.WorkflowID) == "" ||
			strings.TrimSpace(status.WorkflowNamespace) == "" {
			_ = workflow.Sleep(ctx, fcafPreconditionPollInterval)
			continue
		}

		var closedResult workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(ctx, checkAct.Name(), workflowengine.ActivityInput{
			Payload: activities.CheckWorkflowClosedActivityInput{
				WorkflowID:        status.WorkflowID,
				RunID:             status.RunID,
				WorkflowNamespace: status.WorkflowNamespace,
			},
		}).Get(ctx, &closedResult); err != nil {
			return workflowengine.WorkflowResult{}, err
		}
		closed, err := decodeCheckWorkflowClosed(closedResult.Output)
		if err != nil {
			return workflowengine.WorkflowResult{}, err
		}
		if !closed.Closed {
			_ = workflow.Sleep(ctx, fcafPreconditionPollInterval)
			continue
		}

		return w.fetchQueuedPipelineResult(ctx, getResultAct, status)
	}
}

func decodeEnqueuePipelineRunTicketOutput(
	raw any,
) (activities.EnqueuePipelineRunTicketActivityOutput, error) {
	if output, ok := raw.(activities.EnqueuePipelineRunTicketActivityOutput); ok {
		return output, nil
	}
	var output activities.EnqueuePipelineRunTicketActivityOutput
	data, err := json.Marshal(raw)
	if err != nil {
		return output, fmt.Errorf("marshal enqueue ticket output: %w", err)
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return output, fmt.Errorf("decode enqueue ticket output: %w", err)
	}
	return output, nil
}

func enqueueRunTicketStatusToSemaphoreStatus(
	ticketID string,
	runnerIDs []string,
	output activities.EnqueuePipelineRunTicketActivityOutput,
) mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView {
	leaderRunnerID := ""
	if len(runnerIDs) > 0 {
		leaderRunnerID = runnerIDs[0]
	}
	return mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView{
		TicketID:          ticketID,
		Status:            output.Status,
		Position:          output.Position,
		LineLen:           output.LineLen,
		LeaderRunnerID:    leaderRunnerID,
		RequiredRunnerIDs: append([]string(nil), runnerIDs...),
		WorkflowID:        output.WorkflowID,
		RunID:             output.RunID,
		WorkflowNamespace: output.WorkflowNamespace,
		ErrorMessage:      output.ErrorMessage,
	}
}

func (w *FCAFPreconditionPipelineWorkflow) fetchQueuedPipelineResult(
	ctx workflow.Context,
	getResultAct *activities.GetWorkflowResultActivity,
	status mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView,
) (workflowengine.WorkflowResult, error) {
	if strings.TrimSpace(status.WorkflowID) == "" ||
		strings.TrimSpace(status.WorkflowNamespace) == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewAppError(
			workflowengine.WorkflowError{
				Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
				Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
				Message: firstNonEmpty(
					status.ErrorMessage,
					"fcaf precondition pipeline did not start",
				),
			},
		)
	}

	var resultResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, getResultAct.Name(), workflowengine.ActivityInput{
		Payload: activities.GetWorkflowResultActivityInput{
			WorkflowID:           status.WorkflowID,
			RunID:                status.RunID,
			WorkflowNamespace:    status.WorkflowNamespace,
			ReturnFailureDetails: true,
		},
	}).Get(ctx, &resultResult); err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	result, err := decodeWorkflowResult(resultResult.Output)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	pipelineResult, err := fcafPipelineExecutionResult(result, status.WorkflowID, status.RunID)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	return workflowengine.WorkflowResult{
		Message: "FCAF precondition pipeline completed",
		Output:  pipelineResult,
	}, nil
}

func failedPipelineWorkflowResult(err error) (workflowengine.WorkflowResult, bool) {
	failure := workflowengine.ParseWorkflowError(err)
	if failure.Details == nil {
		return workflowengine.WorkflowResult{}, false
	}
	output, hasOutput := failure.Details["output"]
	errors, hasErrors := failure.Details["errors"]
	if !hasOutput && !hasErrors {
		return workflowengine.WorkflowResult{}, false
	}
	return workflowengine.WorkflowResult{
		WorkflowID:    failure.WorkflowID,
		WorkflowRunID: failure.RunID,
		Output:        output,
		Errors:        errors,
	}, true
}

func fcafPipelineExecutionResult(
	result workflowengine.WorkflowResult,
	workflowID string,
	runID string,
) (evidence.PipelineExecutionResult, error) {
	failures, err := evidence.DecodePipelineStepFailures(result.Errors)
	if err != nil {
		return evidence.PipelineExecutionResult{}, err
	}
	if strings.TrimSpace(result.WorkflowID) != "" {
		workflowID = result.WorkflowID
	}
	if strings.TrimSpace(result.WorkflowRunID) != "" {
		runID = result.WorkflowRunID
	}
	return evidence.PipelineExecutionResult{
		Output:        result.Output,
		WorkflowID:    workflowID,
		WorkflowRunID: runID,
		StepFailures:  failures,
	}, nil
}

func fcafPreconditionWorkflowID(parentWorkflowID string, preconditionID string) string {
	return fmt.Sprintf(
		"%s-%s",
		strings.TrimSpace(parentWorkflowID),
		canonify.CanonifyPlain(strings.TrimPrefix(strings.TrimSpace(preconditionID), "pipeline.")),
	)
}

func fcafTestsWorkflowID(parentWorkflowID string) string {
	return fmt.Sprintf(
		"%s-tests",
		strings.TrimSpace(parentWorkflowID),
	)
}

func fcafSingleTestWorkflowID(parentWorkflowID string, testID string) string {
	return fmt.Sprintf(
		"%s-test-%s",
		strings.TrimSpace(parentWorkflowID),
		canonify.CanonifyPlain(testID),
	)
}

func addFCAFPipelineReference(
	references []FCAFPipelineResultReference,
	precondition dsl.PreconditionDefinition,
	result evidence.PipelineExecutionResult,
	namespace string,
) []FCAFPipelineResultReference {
	aliases := []string{precondition.ID}
	if pipelineID := strings.TrimSpace(precondition.PipelineID); pipelineID != "" {
		aliases = append(aliases, pipelineID)
	}
	for index := range references {
		if references[index].WorkflowID != result.WorkflowID ||
			references[index].RunID != result.WorkflowRunID ||
			references[index].Namespace != namespace {
			continue
		}
		references[index].Aliases = appendUniqueStrings(references[index].Aliases, aliases...)
		return references
	}
	return append(references, FCAFPipelineResultReference{
		Aliases:    aliases,
		WorkflowID: result.WorkflowID,
		RunID:      result.WorkflowRunID,
		Namespace:  namespace,
	})
}

func filterFCAFPipelineReferences(
	references []FCAFPipelineResultReference,
	requiredPreconditions []string,
) []FCAFPipelineResultReference {
	if len(requiredPreconditions) == 0 {
		return append([]FCAFPipelineResultReference(nil), references...)
	}
	required := make(map[string]struct{}, len(requiredPreconditions))
	for _, id := range requiredPreconditions {
		required[id] = struct{}{}
	}
	filtered := make([]FCAFPipelineResultReference, 0, len(references))
	for _, reference := range references {
		for _, alias := range reference.Aliases {
			if _, ok := required[alias]; ok {
				filtered = append(filtered, reference)
				break
			}
		}
	}
	return filtered
}

func filterFCAFTestPipelineReferences(
	references []FCAFPipelineResultReference,
	testPreconditions map[string][]string,
	requiredPreconditions []string,
) []FCAFPipelineResultReference {
	if testPreconditions == nil {
		return filterFCAFPipelineReferences(references, requiredPreconditions)
	}
	if len(requiredPreconditions) == 0 {
		return nil
	}
	return filterFCAFPipelineReferences(references, requiredPreconditions)
}

func normalizedFCAFTestIDs(payload FCAFTestWorkflowPayload) []string {
	if len(payload.TestIDs) > 0 {
		return append([]string(nil), payload.TestIDs...)
	}
	testID := strings.TrimSpace(payload.TestID)
	if testID == "" {
		return nil
	}
	return []string{testID}
}

func mergeFCAFPublicReport(target *engine.Report, source engine.Report) {
	if target.Evidence == nil {
		target.Evidence = engine.EvidenceMap{}
	}
	seenTests := make(map[string]struct{}, len(target.ExecutedTests)+len(source.ExecutedTests))
	for _, test := range target.ExecutedTests {
		seenTests[test.TestID] = struct{}{}
	}
	for _, test := range source.ExecutedTests {
		if _, exists := seenTests[test.TestID]; exists {
			continue
		}
		seenTests[test.TestID] = struct{}{}
		target.ExecutedTests = append(target.ExecutedTests, test)
	}
	for key, value := range source.Evidence {
		if _, exists := target.Evidence[key]; !exists {
			target.Evidence[key] = value
		}
	}
	target.Summary = fcafSummaryFromExecutedTests(target.ExecutedTests)
}

func keepOnlyFCAFExecutedTests(report *engine.Report, testIDs []string) {
	if report == nil {
		return
	}
	keep := make(map[string]struct{}, len(testIDs))
	for _, testID := range testIDs {
		keep[testID] = struct{}{}
	}
	filtered := report.ExecutedTests[:0]
	for _, test := range report.ExecutedTests {
		if _, ok := keep[test.TestID]; ok {
			filtered = append(filtered, test)
		}
	}
	report.ExecutedTests = filtered
	report.SelectedTestIDs = append([]string(nil), testIDs...)
	report.Summary = fcafSummaryFromExecutedTests(report.ExecutedTests)
	report.Status = fcafAggregateStatus(report.ExecutedTests)
}

func stripFCAFEvidenceValues(report *engine.Report) {
	if report == nil {
		return
	}
	for key, record := range report.Evidence {
		if record.Type == "pipeline.run" {
			continue
		}
		record.Value = nil
		report.Evidence[key] = record
	}
}

func fcafSummaryFromExecutedTests(tests []engine.ExecutedTest) engine.Summary {
	summary := engine.Summary{}
	for _, test := range tests {
		switch test.Status {
		case "passed":
			summary.Pass++
		case "failed":
			summary.Fail++
		case "blocked":
			summary.Blocked++
		case "skipped":
			summary.Skipped++
		case "inconclusive":
			summary.Inconclusive++
		case "not_applicable":
			summary.NotApplicable++
		default:
			summary.Error++
		}
	}
	return summary
}

func sortedFCAFEvidenceKeys(items engine.EvidenceMap) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func fcafAggregateStatus(tests []engine.ExecutedTest) string {
	for _, test := range tests {
		if test.Status != "passed" {
			return "failed"
		}
	}
	return "passed"
}

func appendUniqueStrings(existing []string, values ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(values))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		existing = append(existing, value)
	}
	return existing
}

func fcafQueueTicketID(parentWorkflowID string) string {
	return fmt.Sprintf("fcaf-%s-%s", canonify.CanonifyPlain(parentWorkflowID), uuid.NewString())
}

func decodeFCAFAssessmentActivityOutput(raw any) (activities.FCAFAssessmentActivityOutput, error) {
	if output, ok := raw.(activities.FCAFAssessmentActivityOutput); ok {
		return output, nil
	}
	var output activities.FCAFAssessmentActivityOutput
	data, err := json.Marshal(raw)
	if err != nil {
		return output, fmt.Errorf("marshal FCAF activity output: %w", err)
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return output, fmt.Errorf("decode FCAF activity output: %w", err)
	}
	return output, nil
}

func summarizeFCAFFailures(report engine.Report) string {
	if len(report.Failures) == 0 {
		report.PopulateFailures()
	}
	if len(report.Failures) == 0 {
		return summarizeExecutedFCAFFailures(report.ExecutedTests)
	}
	lines := make([]string, 0, len(report.Failures)+1)
	lines = append(lines, fmt.Sprintf("%d FCAF test(s) did not pass", len(report.Failures)))
	for _, failure := range report.Failures {
		reason := failure.Message
		if len(failure.Reasons) > 0 {
			first := failure.Reasons[0]
			reason = fmt.Sprintf("%s %s: %s", first.Scope, first.ID, first.Message)
		}
		lines = append(lines, fmt.Sprintf("- %s (%s): %s", failure.TestID, failure.Status, reason))
	}
	return strings.Join(lines, "\n")
}

func summarizeExecutedFCAFFailures(tests []engine.ExecutedTest) string {
	failed := make([]engine.ExecutedTest, 0)
	for _, test := range tests {
		if test.Status != "passed" {
			failed = append(failed, test)
		}
	}
	lines := make([]string, 0, len(failed)+1)
	lines = append(lines, fmt.Sprintf("%d FCAF test(s) did not pass", len(failed)))
	for _, test := range failed {
		reason := test.Outcome.Reason
		for _, precondition := range test.Preconditions {
			if precondition.Status != "passed" {
				reason = fmt.Sprintf(
					"precondition %s: %s",
					precondition.ID,
					precondition.Message,
				)
				break
			}
		}
		if reason == "" {
			for _, assertion := range test.Assertions {
				if assertion.Status != "passed" {
					reason = fmt.Sprintf("assertion %s: %s", assertion.ID, assertion.Message)
					break
				}
			}
		}
		lines = append(lines, fmt.Sprintf("- %s (%s): %s", test.TestID, test.Status, reason))
	}
	return strings.Join(lines, "\n")
}

func decodeFCAFExecutionPlan(raw any) (activities.FCAFResolveExecutionPlanActivityOutput, error) {
	if output, ok := raw.(activities.FCAFResolveExecutionPlanActivityOutput); ok {
		return output, nil
	}
	var output activities.FCAFResolveExecutionPlanActivityOutput
	data, err := json.Marshal(raw)
	if err != nil {
		return output, fmt.Errorf("marshal FCAF execution plan: %w", err)
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return output, fmt.Errorf("decode FCAF execution plan: %w", err)
	}
	return output, nil
}

func decodeSemaphoreRunStatus(
	raw any,
) (mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView, error) {
	if output, ok := raw.(mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView); ok {
		return output, nil
	}
	var output mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView
	data, err := json.Marshal(raw)
	if err != nil {
		return output, fmt.Errorf("marshal semaphore status: %w", err)
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return output, fmt.Errorf("decode semaphore status: %w", err)
	}
	return output, nil
}

func decodeCheckWorkflowClosed(raw any) (activities.CheckWorkflowClosedActivityOutput, error) {
	if output, ok := raw.(activities.CheckWorkflowClosedActivityOutput); ok {
		return output, nil
	}
	var output activities.CheckWorkflowClosedActivityOutput
	data, err := json.Marshal(raw)
	if err != nil {
		return output, fmt.Errorf("marshal workflow status: %w", err)
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return output, fmt.Errorf("decode workflow status: %w", err)
	}
	return output, nil
}

func decodeWorkflowResult(raw any) (workflowengine.WorkflowResult, error) {
	if output, ok := raw.(workflowengine.WorkflowResult); ok {
		return output, nil
	}
	var output workflowengine.WorkflowResult
	data, err := json.Marshal(raw)
	if err != nil {
		return output, fmt.Errorf("marshal workflow result: %w", err)
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return output, fmt.Errorf("decode workflow result: %w", err)
	}
	return output, nil
}

func (w *FCAFAssessmentWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "FCAFAssessment-" + uuid.NewString(),
		TaskQueue:                FCAFAssessmentTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}
	return fcafStartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}

type fcafPipelineRunnerInfo struct {
	RunnerIDs         []string
	NeedsGlobalRunner bool
}

func fetchFCAFPreconditionPipelineYAML(
	ctx workflow.Context,
	appURL string,
	pipelineID string,
) (string, error) {
	act := activities.NewInternalHTTPActivity()
	var result workflowengine.ActivityResult
	err := workflow.ExecuteActivity(ctx, act.Name(), workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodGet,
			URL: utils.JoinURL(
				appURL,
				"api", "pipeline", "get-yaml",
			) + "?pipeline_identifier=" + pipelineID,
			ExpectedStatus: http.StatusOK,
		},
	}).Get(ctx, &result)
	if err != nil {
		return "", err
	}
	return decodePipelineYAMLHTTPOutput(result.Output)
}

func decodePipelineYAMLHTTPOutput(raw any) (string, error) {
	if body, ok := raw.(string); ok && body != "" {
		return body, nil
	}

	if outputMap, ok := raw.(map[string]any); ok {
		body, ok := outputMap["body"]
		if !ok {
			return "", fmt.Errorf("missing body in pipeline yaml response")
		}
		if bodyStr, ok := body.(string); ok && bodyStr != "" {
			return bodyStr, nil
		}
		return "", fmt.Errorf("unexpected pipeline yaml body type %T", body)
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("marshal pipeline yaml output: %w", err)
	}

	var outputMap map[string]any
	if err := json.Unmarshal(data, &outputMap); err == nil {
		if body, ok := outputMap["body"].(string); ok && body != "" {
			return body, nil
		}
	}

	var output string
	if err := json.Unmarshal(data, &output); err == nil && output != "" {
		return output, nil
	}

	return "", fmt.Errorf("unexpected pipeline yaml output type %T", raw)
}

func prepareFCAFPreconditionPipelineYAML(
	pipelineYAML string,
	runtime map[string]any,
) (string, fcafPipelineRunnerInfo, error) {
	def, err := pipelineinternal.ParseWorkflow(pipelineYAML)
	if err != nil {
		return "", fcafPipelineRunnerInfo{}, err
	}
	info := collectFCAFRunnerInfo(def.Steps)

	runnerID := lookupStringMap(runtime, "runner_id")
	if info.NeedsGlobalRunner {
		if strings.TrimSpace(runnerID) == "" {
			return "", fcafPipelineRunnerInfo{}, fmt.Errorf(
				"runner_id is required for mobile-automation pipeline preconditions",
			)
		}
		if len(info.RunnerIDs) > 0 {
			return "", fcafPipelineRunnerInfo{}, fmt.Errorf(
				"runner_id cannot be combined with step runner_id",
			)
		}
		def.Runtime.GlobalRunnerID = runnerID
	}

	rewritten, err := yaml.Marshal(def)
	if err != nil {
		return "", fcafPipelineRunnerInfo{}, err
	}
	return string(rewritten), info, nil
}

func mergeSemaphoreStatus(
	previous mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView,
	current mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView,
) mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView {
	if strings.TrimSpace(current.WorkflowID) == "" {
		current.WorkflowID = previous.WorkflowID
	}
	if strings.TrimSpace(current.RunID) == "" {
		current.RunID = previous.RunID
	}
	if strings.TrimSpace(current.WorkflowNamespace) == "" {
		current.WorkflowNamespace = previous.WorkflowNamespace
	}
	if strings.TrimSpace(current.ErrorMessage) == "" {
		current.ErrorMessage = previous.ErrorMessage
	}
	if strings.TrimSpace(current.LeaderRunnerID) == "" {
		current.LeaderRunnerID = previous.LeaderRunnerID
	}
	if len(current.RequiredRunnerIDs) == 0 {
		current.RequiredRunnerIDs = append([]string(nil), previous.RequiredRunnerIDs...)
	}
	return current
}

func isTerminalSemaphoreStatus(status mobilerunnersemaphore.MobileRunnerSemaphoreRunStatus) bool {
	switch status {
	case mobilerunnersemaphore.MobileRunnerSemaphoreRunFailed,
		mobilerunnersemaphore.MobileRunnerSemaphoreRunCanceled,
		mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound:
		return true
	default:
		return false
	}
}

func collectFCAFRunnerInfo(steps []pipelineinternal.StepDefinition) fcafPipelineRunnerInfo {
	runnerSet := map[string]struct{}{}
	needsGlobal := false
	var walk func([]pipelineinternal.StepDefinition)
	walk = func(steps []pipelineinternal.StepDefinition) {
		for _, step := range steps {
			runnerID := ""
			if step.With.Payload != nil {
				if raw, ok := step.With.Payload["runner_id"].(string); ok {
					runnerID = strings.TrimSpace(raw)
				}
			}
			if step.Use == "mobile-automation" && runnerID == "" {
				needsGlobal = true
			}
			if runnerID != "" {
				runnerSet[runnerID] = struct{}{}
			}
			if len(step.OnError) > 0 {
				next := make([]pipelineinternal.StepDefinition, 0, len(step.OnError))
				for _, child := range step.OnError {
					if child == nil {
						continue
					}
					next = append(next, pipelineinternal.StepDefinition{StepSpec: child.StepSpec})
				}
				walk(next)
			}
			if len(step.OnSuccess) > 0 {
				next := make([]pipelineinternal.StepDefinition, 0, len(step.OnSuccess))
				for _, child := range step.OnSuccess {
					if child == nil {
						continue
					}
					next = append(next, pipelineinternal.StepDefinition{StepSpec: child.StepSpec})
				}
				walk(next)
			}
		}
	}
	walk(steps)

	runnerIDs := make([]string, 0, len(runnerSet))
	for runnerID := range runnerSet {
		runnerIDs = append(runnerIDs, runnerID)
	}
	sort.Strings(runnerIDs)
	return fcafPipelineRunnerInfo{
		RunnerIDs:         runnerIDs,
		NeedsGlobalRunner: needsGlobal,
	}
}

func fcafRunnerIDsWithGlobal(info fcafPipelineRunnerInfo, globalRunnerID string) []string {
	runnerIDs := append([]string{}, info.RunnerIDs...)
	globalRunnerID = strings.TrimSpace(globalRunnerID)
	if info.NeedsGlobalRunner && globalRunnerID != "" {
		found := false
		for _, runnerID := range runnerIDs {
			if runnerID == globalRunnerID {
				found = true
				break
			}
		}
		if !found {
			runnerIDs = append(runnerIDs, globalRunnerID)
		}
	}
	sort.Strings(runnerIDs)
	return runnerIDs
}

func pipelineNamespaceFromID(pipelineID string) (string, error) {
	trimmed := strings.Trim(strings.TrimSpace(pipelineID), "/")
	if trimmed == "" {
		return "", fmt.Errorf("pipeline_id is empty")
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" {
		return "", fmt.Errorf(
			"pipeline_id %q must be in /org-owner/canonified-name form",
			pipelineID,
		)
	}
	return parts[0], nil
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func lookupStringMap(in map[string]any, key string) string {
	if in == nil {
		return ""
	}
	value, _ := in[key].(string)
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
