// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/temporalid"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/proto"
)

func ExecuteStep(
	id string,
	use string,
	with pipeline.StepInputs,
	activityOptions *pipeline.ActivityOptionsConfig,
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	errCode := errorcodes.Codes[errorcodes.PipelineInputError]
	s := &pipeline.StepDefinition{
		StepSpec: pipeline.StepSpec{
			ID:              id,
			Use:             use,
			With:            with,
			ActivityOptions: activityOptions,
			Metadata:        nil,
		},
		ContinueOnError: false,
	}

	step := registry.Registry[s.Use]
	stepGlobalCfg := globalCfg
	if step.Kind == registry.TaskActivity {
		stepGlobalCfg = selectConfigKeys(globalCfg, step.InheritedConfigKeys)
	}
	err := pipeline.ResolveInputs(s, stepGlobalCfg, dataCtx)
	if err != nil {
		appErr := workflowengine.NewNonRetryableAppError(
			workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: fmt.Sprintf("error resolving inputs for step %s: %s", s.ID, err.Error()),
			},
		)

		return nil, appErr
	}
	switch step.Kind {
	case registry.TaskActivity:
		payload, err := DecodePayload(s)
		if err != nil {
			appErr := workflowengine.NewNonRetryableAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf(
						"error decoding payload for step %s: %s",
						s.ID,
						err.Error(),
					),
				},
			)

			return nil, appErr
		}
		actAO := ao
		actAO.Summary = summaryLine(s.ID + " (" + s.Use + ")")
		ctx = workflow.WithActivityOptions(ctx, actAO)
		act := step.NewFunc(nil).(workflowengine.Activity)
		input := workflowengine.ActivityInput{
			Payload: payload,
			Config:  workflowengine.StringifyConfig(s.With.Config),
		}
		var result workflowengine.ActivityResult

		if s.Use == "email" {
			cfgAct := act.(workflowengine.ConfigurableActivity)

			if err := cfgAct.Configure(&input); err != nil {
				appErr := workflowengine.NewNonRetryableAppError(
					workflowengine.WorkflowError{
						Code:    errCode.Code,
						Summary: errCode.Description,
						Message: fmt.Sprintf(
							"error configuring activity %s: %s",
							s.ID,
							err.Error(),
						),
					},
				)

				return result, appErr
			}
		}
		execAct, ok := act.(workflowengine.ExecutableActivity)
		if !ok {
			appErr := workflowengine.NewNonRetryableAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf("activity %s is not executable", s.ID),
				},
			)

			return result, appErr
		}

		if err := ensureStepInputSize(s.ID, input); err != nil {
			return nil, err
		}
		err = workflow.ExecuteActivity(ctx, execAct.Name(), input).Get(ctx, &result)
		if err != nil {
			return result, err
		}

		return workflowengine.StepOutputFromActivityResult(step.OutputKind, result), nil
	case registry.TaskWorkflow:
		payload, err := DecodePayload(s)
		if err != nil {
			appErr := workflowengine.NewNonRetryableAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf(
						"error decoding payload for step %s: %s",
						s.ID,
						err.Error(),
					),
				},
			)

			return nil, appErr
		}
		w := step.NewFunc(nil).(workflowengine.Workflow)
		input := workflowengine.WorkflowInput{
			Payload:         payload,
			Config:          workflowengine.MergeTelemetryConfig(ctx, s.With.Config),
			ActivityOptions: &ao,
		}

		if err := ensureStepInputSize(s.ID, input); err != nil {
			return nil, err
		}
		if s.Use == mobileAutomationStepUse {
			return runMobileStep(ctx, s, input)
		}

		var memo map[string]any
		memo, _ = input.Config["memo"].(map[string]any)

		var result workflowengine.WorkflowResult
		opts := workflow.ChildWorkflowOptions{
			WorkflowID:        taskWorkflowID(ctx, s.ID),
			TaskQueue:         PipelineTaskQueue,
			ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
			Memo:              memo,
			StaticSummary:     summaryLine(s.ID),
		}
		ctxChild := workflow.WithChildOptions(ctx, opts)
		err = workflow.ExecuteChildWorkflow(
			ctxChild,
			w.Name(),
			input,
		).Get(ctxChild, &result)
		if err != nil {
			return failedTaskWorkflowOutput(err), err
		}
		return result.Output, nil
	}

	return nil, nil
}

// runMobileStep runs a mobile-automation step's workflow in the runner
// namespace of its device, through the Run step operation of the runner's
// Nexus endpoint.
func runMobileStep(
	ctx workflow.Context,
	s *pipeline.StepDefinition,
	input workflowengine.WorkflowInput,
) (any, error) {
	namespace, _ := s.With.Config[runnerNamespaceConfigKey].(string)
	if strings.TrimSpace(namespace) == "" {
		return nil, missingMobileStepConfig(s.ID, runnerNamespaceConfigKey)
	}
	pipelineRun, err := workflowengine.DecodePayload[mobiledevice.PipelineRun](
		s.With.Config[pipelineRunConfigKey],
	)
	if err != nil || pipelineRun.WorkflowID == "" {
		return nil, missingMobileStepConfig(s.ID, pipelineRunConfigKey)
	}
	executionTimeout, err := time.ParseDuration(
		workflowengine.AsString(s.With.Config[pipelineExecutionTimeoutConfigKey]),
	)
	if err != nil {
		return nil, missingMobileStepConfig(s.ID, pipelineExecutionTimeoutConfigKey)
	}
	var result workflowengine.WorkflowResult
	err = workflow.NewNexusClient(namespace, mobiledevice.ServiceName).ExecuteOperation(
		ctx,
		mobiledevice.RunStepOperationName,
		mobiledevice.RunStepInput{
			WorkflowID:       taskWorkflowID(ctx, s.ID),
			ExecutionTimeout: executionTimeout,
			Pipeline:         pipelineRun,
			// The runner owner reads this input, so only what the step workflow needs is sent.
			Input: workflowengine.WorkflowInput{
				Payload:         input.Payload,
				Config:          selectConfigKeys(input.Config, runnerStepConfigKeys),
				ActivityOptions: input.ActivityOptions,
			},
		},
		workflow.NexusOperationOptions{
			ScheduleToCloseTimeout: executionTimeout,
			ScheduleToStartTimeout: deviceOperationScheduleToStart,
			Summary:                summaryLine(s.ID),
		},
	).Get(ctx, &result)
	if err != nil {
		return failedTaskWorkflowOutput(err), err
	}
	return result.Output, nil
}

// missingMobileStepConfig fails a mobile step whose setup did not set key.
func missingMobileStepConfig(stepID, key string) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
	return workflowengine.NewNonRetryableAppError(workflowengine.WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: fmt.Sprintf("missing %s for step %s", key, stepID),
	})
}

// taskWorkflowID is the workflow ID of a task workflow step: the running workflow's ID and
// the step ID, plus the attempt inside a Linked Steps Workflow, whose attempts would
// otherwise reuse one workflow ID.
func taskWorkflowID(ctx workflow.Context, stepID string) string {
	info := workflow.GetInfo(ctx)
	id := fmt.Sprintf("%s-%s", info.WorkflowExecution.ID, canonify.CanonifyPlain(stepID))
	if info.WorkflowType.Name == pipeline.LinkedStepsWorkflowName {
		id = fmt.Sprintf("%s-a%d", id, info.Attempt)
	}
	return temporalid.Bounded(id)
}

func Execute(
	s *pipeline.StepDefinition,
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	with := s.With
	with.Config = maps.Clone(s.With.Config)
	if with.Config == nil {
		with.Config = map[string]any{}
	}
	with.Config[workflowengine.StepIDConfigKey] = s.ID
	return ExecuteStep(s.ID, s.Use, with, s.ActivityOptions, ctx, globalCfg, dataCtx, ao)
}

// failedTaskWorkflowOutput returns what a step keeps when its task workflow
// failed: the output it reported in its failure Details.output, such as the
// screenshots a failed mobile flow took, or the zero WorkflowResult.
func failedTaskWorkflowOutput(err error) any {
	if out := workflowengine.ExtractOutputFromError(err); out != nil {
		return out
	}
	return workflowengine.WorkflowResult{}
}

// maxStepInputBytes keeps a step's scheduling command below the 4 MiB gRPC message limit
// the Temporal frontend enforces on workflow task completions.
const maxStepInputBytes = 3 << 20

const (
	// payloadFramingBytes bounds the protobuf framing and metadata around the JSON data.
	payloadFramingBytes = 1 << 10
	// secretsEnvelopeBytes bounds the fixed part of one encrypted secrets envelope:
	// its keys, algorithm, nonce and GCM tag.
	secretsEnvelopeBytes = 256
)

// ensureStepInputSize rejects a step input that would make the workflow task completion
// exceed the Temporal gRPC message limit, which would otherwise wedge the run. The size is
// measured on the payload the data converter produces, including encrypted secrets.
func ensureStepInputSize(stepID string, input any) error {
	size, encodeErr := encodedStepInputSize(input)
	if encodeErr != nil {
		// An encoding error is left to the SDK, which reports it when it encodes the input.
		return nil //nolint:nilerr // the SDK reports encoding errors itself
	}
	if size <= maxStepInputBytes {
		return nil
	}
	errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
	return workflowengine.NewNonRetryableAppError(workflowengine.WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: fmt.Sprintf(
			"step %s input is %d bytes; the limit is %d bytes because Temporal rejects workflow task messages above 4 MiB",
			stepID,
			size,
			maxStepInputBytes,
		),
	})
}

// encodedStepInputSize returns the size of the payloads the data converter makes from
// input, or a bound that decides the same way. This runs in workflow code, where encoding
// a multi-MiB input can exceed the SDK's one-second deadlock detection timeout, so it
// encodes only inputs close to the limit.
func encodedStepInputSize(input any) (int, error) {
	plain, err := json.Marshal(input)
	if err != nil {
		return 0, err
	}
	// The converter emits this same JSON with each "secrets" value replaced by a larger
	// encrypted envelope, so the plain size is a lower bound.
	if len(plain) > maxStepInputBytes {
		return len(plain), nil
	}
	upper := len(plain) + payloadFramingBytes
	if secrets := bytes.Count(plain, []byte(`"secrets":`)); secrets > 0 {
		// Base64 of the ciphertext grows each secrets value by a third, plus its envelope.
		upper += len(plain)/3 + secrets*secretsEnvelopeBytes
	}
	if upper <= maxStepInputBytes {
		return upper, nil
	}
	payloads, err := temporalcrypto.DataConverter().ToPayloads(input)
	if err != nil {
		return 0, err
	}
	return proto.Size(payloads), nil
}

// summaryLine flattens text to a single line for Temporal UI summaries.
func summaryLine(text string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(text)
}

// selectConfigKeys copies the listed keys that are present in cfg.
func selectConfigKeys(cfg map[string]any, keys []string) map[string]any {
	selected := make(map[string]any, len(keys))
	for _, key := range keys {
		if value, ok := cfg[key]; ok {
			selected[key] = value
		}
	}
	return selected
}

// runnerStepConfigKeys are the config keys the mobile step workflow reads in
// its runner namespace (workflows/mobile.go, BuildWorkflow, the step memo and
// the telemetry the runner activity forwards). Nothing else of the pipeline
// config, such as semaphore tickets, user_mail or github_pr_comment, may reach
// that namespace's history: its runner owner can read it.
var runnerStepConfigKeys = []string{
	workflowengine.AppURLConfigKey,
	"taskqueue",
	mobileExternalInstallConfigKey,
	"run_identifier",
	"runner_id",
	workflowengine.StepIDConfigKey,
	"memo",
	workflowengine.TelemetryRootWorkflowIDKey,
	workflowengine.TelemetryRootRunIDKey,
	workflowengine.TelemetryParentWorkflowIDKey,
	workflowengine.TelemetryParentRunIDKey,
}

func ExecuteOnError(
	s *pipeline.OnErrorStepDefinition,
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	return ExecuteStep(s.ID, s.Use, s.With, s.ActivityOptions, ctx, globalCfg, dataCtx, ao)
}

func ExecuteOnSuccess(
	s *pipeline.OnSuccessStepDefinition,
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	return ExecuteStep(s.ID, s.Use, s.With, s.ActivityOptions, ctx, globalCfg, dataCtx, ao)
}

// runChildPipeline executes a nested child pipeline and returns its outputs. returnOutputs
// lists the child's step outputs the parent references; the child returns only those.
// parentDevices are the devices the parent prepared, which the child reuses.
func runChildPipeline(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	input PipelineWorkflowInput,
	workflowName string,
	dataCtx map[string]any,
	runMetadata *workflowengine.WorkflowRunMetadata,
	returnOutputs []string,
	parentDevices map[string]any,
) (any, error) {
	// Fetch child pipeline YAML
	yaml, err := fetchChildPipelineYAML(ctx, step, input, runMetadata)
	if err != nil {
		return nil, err
	}

	// Parse workflow definition
	wfDef, err := pipeline.ParseWorkflow(yaml)
	if err != nil {
		return nil, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errorcodes.Codes[errorcodes.PipelineParsingError].Code,
					Summary: errorcodes.Codes[errorcodes.PipelineParsingError].Description,
					Message: err.Error(),
				},
			),
			runMetadata,
		)
	}

	memo := map[string]any{"test": wfDef.Name}
	options := PrepareWorkflowOptions(wfDef.Runtime)
	options.Options.Memo = memo

	childOpts := workflow.ChildWorkflowOptions{
		WorkflowID: temporalid.Bounded(fmt.Sprintf(
			"%s-%s-%s",
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			canonify.CanonifyPlain(step.ID),
			wfDef.Name,
		)),
		TaskQueue:         PipelineTaskQueue,
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
		StaticSummary:     summaryLine(step.ID + ": " + wfDef.Name),
	}

	ao := PrepareActivityOptions(options.ActivityOptions, step.ActivityOptions)
	childOpts.RetryPolicy = ao.RetryPolicy
	ctxChild := workflow.WithChildOptions(ctx, childOpts)
	err = pipeline.ResolveInputs(&step, input.WorkflowInput.Config, dataCtx)
	if err != nil {
		return nil, err
	}
	childInput := PipelineWorkflowInput{
		WorkflowDefinition: wfDef,
		WorkflowInput: workflowengine.WorkflowInput{
			Config:          workflowengine.MergeTelemetryConfig(ctx, step.With.Config),
			Payload:         step.With.Payload,
			ActivityOptions: &ao,
		},
		Debug:         wfDef.Runtime.Debug,
		ReturnOutputs: returnOutputs,
		ParentDevices: parentDevices,
	}
	childInput.WorkflowInput.Config[workflowengine.StepIDConfigKey] = step.ID

	if err := ensureStepInputSize(step.ID, childInput); err != nil {
		return nil, err
	}
	var childResult workflowengine.WorkflowResult
	err = workflow.ExecuteChildWorkflow(
		ctxChild,
		workflowName,
		childInput,
	).Get(ctxChild, &childResult)

	if err != nil {
		return nil, err
	}

	return childResult.Output, nil
}

// fetchChildPipelineYAML resolves the child pipeline record and returns its YAML.
func fetchChildPipelineYAML(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	input PipelineWorkflowInput,
	meta *workflowengine.WorkflowRunMetadata,
) (string, error) {
	pipelineID, ok := step.With.Payload["pipeline_id"].(string)
	if !ok || pipelineID == "" {
		return "", workflowengine.NewMissingOrInvalidPayloadError(
			fmt.Errorf("missing pipeline_id"),
			meta,
		)
	}

	namespace, ok := input.WorkflowInput.Config["namespace"].(string)
	if !ok || namespace == "" {
		return "", workflowengine.NewWorkflowError(
			workflowengine.NewMissingConfigError("namespace", meta),
			meta,
		)
	}

	var response workflowengine.ActivityResult
	req := workflowengine.ActivityInput{
		Payload: activities.ResolveRecordInput{
			CanonifiedName: pipelineID,
			Collection:     "pipelines",
			OwnerNamespace: namespace,
		},
	}

	if err := workflow.ExecuteActivity(ctx, activities.ResolveRecordActivityName, req).
		Get(ctx, &response); err != nil {
		return "", workflowengine.NewWorkflowError(err, meta)
	}

	record, _ := response.Output.(map[string]any)
	yaml, ok := record["yaml"].(string)
	if !ok {
		return "", workflowengine.NewWorkflowError(
			workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errorcodes.Codes[errorcodes.UnexpectedActivityOutput].Code,
					Summary: errorcodes.Codes[errorcodes.UnexpectedActivityOutput].Description,
					Message: "invalid resolved pipeline output",
					Details: map[string]any{"payload": response.Output},
				},
			),
			meta,
		)
	}

	return yaml, nil
}

func ExtractPipelineOutput(dataCtx map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range dataCtx {
		if key == "inputs" {
			continue
		}

		if stepOutput, ok := value.(map[string]any); ok {
			if outputs, exists := stepOutput["outputs"]; exists {
				result[key] = map[string]any{"outputs": outputs}
			} else {
				result[key] = map[string]any{"outputs": value}
			}
		} else {
			result[key] = map[string]any{"outputs": value}
		}
	}

	return result
}

func enrichDataContext(
	dataCtx map[string]any,
	pipelineName string,
	pipelineURL string,
	hasErrors bool,
	currentTime string,
) map[string]any {
	enriched := make(map[string]any)

	for k, v := range dataCtx {
		enriched[k] = v
	}

	enriched["pipeline_output"] = ExtractPipelineOutput(dataCtx)

	enriched["pipeline_name"] = pipelineName
	enriched["pipeline_url"] = pipelineURL

	if hasErrors {
		enriched["result"] = resultFailed
	} else {
		enriched["result"] = resultSuccess
	}

	enriched["date"] = currentTime

	return enriched
}
