// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporalnexus"
)

// NewMobileDeviceService returns the Nexus service every runner namespace
// serves: each operation starts one workflow in the runner namespace for a
// pipeline run of an organization namespace.
func NewMobileDeviceService(app core.App) *nexus.Service {
	service := nexus.NewService(mobiledevice.ServiceName)
	err := service.Register(
		temporalnexus.MustNewWorkflowRunOperationWithOptions(
			temporalnexus.WorkflowRunOperationOptions[
				mobiledevice.PrepareInput,
				mobiledevice.PrepareResult,
			]{
				Name: mobiledevice.PrepareOperationName,
				Handler: func(
					ctx context.Context,
					in mobiledevice.PrepareInput,
					options nexus.StartOperationOptions,
				) (temporalnexus.WorkflowHandle[mobiledevice.PrepareResult], error) {
					if err := requireCredimiCaller(
						ctx,
						app,
						describeOrganizationWorkflow,
						options,
						mobiledevice.PrepareOperationName,
						temporalnexus.GetOperationInfo(ctx).Namespace,
						in.Pipeline,
						in.WorkflowID,
						in.DeviceID,
					); err != nil {
						return nil, err
					}
					return temporalnexus.ExecuteUntypedWorkflow[mobiledevice.PrepareResult](
						ctx,
						options,
						deviceWorkflowOptions(
							in.WorkflowID,
							in.ExecutionTimeout,
							in.Pipeline,
							nil,
							"device "+in.DeviceID,
						),
						mobiledevice.PrepareWorkflowName,
						in,
					)
				},
			},
		),
		temporalnexus.MustNewWorkflowRunOperationWithOptions(
			temporalnexus.WorkflowRunOperationOptions[
				mobiledevice.RunStepInput,
				workflowengine.WorkflowResult,
			]{
				Name: mobiledevice.RunStepOperationName,
				Handler: func(
					ctx context.Context,
					in mobiledevice.RunStepInput,
					options nexus.StartOperationOptions,
				) (temporalnexus.WorkflowHandle[workflowengine.WorkflowResult], error) {
					deviceID, _ := workflowengine.AsMap(in.Input.Payload)["device_id"].(string)
					if err := requireCredimiCaller(
						ctx,
						app,
						describeOrganizationWorkflow,
						options,
						mobiledevice.RunStepOperationName,
						temporalnexus.GetOperationInfo(ctx).Namespace,
						in.Pipeline,
						in.WorkflowID,
						deviceID,
					); err != nil {
						return nil, err
					}
					memo, _ := in.Input.Config["memo"].(map[string]any)
					return temporalnexus.ExecuteUntypedWorkflow[workflowengine.WorkflowResult](
						ctx,
						options,
						deviceWorkflowOptions(
							in.WorkflowID,
							in.ExecutionTimeout,
							in.Pipeline,
							memo,
							workflowengine.AsString(
								in.Input.Config[workflowengine.StepIDConfigKey],
							),
						),
						mobiledevice.StepWorkflowName,
						in.Input,
					)
				},
			},
		),
		temporalnexus.MustNewWorkflowRunOperationWithOptions(
			temporalnexus.WorkflowRunOperationOptions[
				mobiledevice.CleanUpInput,
				mobiledevice.CleanUpResult,
			]{
				Name: mobiledevice.CleanUpOperationName,
				Handler: func(
					ctx context.Context,
					in mobiledevice.CleanUpInput,
					options nexus.StartOperationOptions,
				) (temporalnexus.WorkflowHandle[mobiledevice.CleanUpResult], error) {
					if err := requireCredimiCaller(
						ctx,
						app,
						describeOrganizationWorkflow,
						options,
						mobiledevice.CleanUpOperationName,
						temporalnexus.GetOperationInfo(ctx).Namespace,
						in.Pipeline,
						in.WorkflowID,
						in.DeviceID,
					); err != nil {
						return nil, err
					}
					return temporalnexus.ExecuteUntypedWorkflow[mobiledevice.CleanUpResult](
						ctx,
						options,
						deviceWorkflowOptions(
							in.WorkflowID,
							in.ExecutionTimeout,
							in.Pipeline,
							nil,
							"device "+in.DeviceID,
						),
						mobiledevice.CleanUpWorkflowName,
						in,
					)
				},
			},
		),
	)
	if err != nil {
		panic(err)
	}
	return service
}

// deviceWorkflowOptions starts a workflow of the mobile device service. Its
// memo names the pipeline run, after the step's own memo entries.
func deviceWorkflowOptions(
	workflowID string,
	executionTimeout time.Duration,
	pipelineRun mobiledevice.PipelineRun,
	memo map[string]any,
	summary string,
) client.StartWorkflowOptions {
	fullMemo := make(map[string]any, len(memo)+1)
	for key, value := range memo {
		fullMemo[key] = value
	}
	fullMemo[mobiledevice.PipelineRunMemoKey] = pipelineRun
	return client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                PipelineTaskQueue,
		Memo:                     fullMemo,
		WorkflowExecutionTimeout: executionTimeout,
		StaticSummary:            summaryLine(summary),
	}
}

// organizationWorkflowDescriber describes a workflow run of an organization
// namespace.
type organizationWorkflowDescriber func(
	ctx context.Context,
	namespace, workflowID, runID string,
) (*workflowservice.DescribeWorkflowExecutionResponse, error)

func describeOrganizationWorkflow(
	ctx context.Context,
	namespace, workflowID, runID string,
) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil {
		return nil, err
	}
	return temporalclient.Describe(ctx, c, workflowID, runID)
}

// requireCredimiCaller accepts an operation only from a Credimi pipeline: the
// caller's link names the pipeline run of the input in its organization
// namespace (or, for a step, a linked-steps block of that run), the workflow
// ID of a device operation is the one Credimi computes for the run, and the
// device is hosted by the runner of handlerNamespace, the namespace serving
// the operation. Runner owners can write to their runner namespace, and only
// Credimi writes to organization namespaces. The server refuses Nexus headers
// that would replace the caller link (component.nexusoperations.disallowedHeaders).
func requireCredimiCaller(
	ctx context.Context,
	app core.App,
	describe organizationWorkflowDescriber,
	options nexus.StartOperationOptions,
	operation string,
	handlerNamespace string,
	pipelineRun mobiledevice.PipelineRun,
	workflowID string,
	deviceID string,
) error {
	if len(options.Links) == 0 {
		return notOrganizationPipeline()
	}
	link, err := temporalnexus.ConvertNexusLinkToLinkWorkflowEvent(options.Links[0])
	if err != nil {
		return notOrganizationPipeline()
	}
	namespace := link.GetNamespace()
	if namespace == "" ||
		strings.HasPrefix(namespace, canonify.RunnerNamespacePrefix) ||
		namespace != pipelineRun.Namespace {
		return notOrganizationPipeline()
	}
	if _, err := pbutils.FindOrganizationByNamespace(app, namespace); err != nil {
		return notOrganizationPipeline()
	}
	if pipelineRun.WorkflowID == "" || pipelineRun.RunID == "" {
		return notOrganizationPipeline()
	}
	if err := requirePipelineRunCaller(
		ctx, describe, operation, pipelineRun, link.GetWorkflowId(), link.GetRunId(),
	); err != nil {
		return err
	}
	var expectedID string
	switch operation {
	case mobiledevice.PrepareOperationName:
		expectedID = mobiledevice.PrepareWorkflowID(pipelineRun.WorkflowID, deviceID)
	case mobiledevice.CleanUpOperationName:
		expectedID = mobiledevice.CleanUpWorkflowID(pipelineRun.WorkflowID, deviceID)
	}
	if expectedID != "" && workflowID != expectedID {
		return nexus.NewHandlerErrorf(
			nexus.HandlerErrorTypeBadRequest,
			"workflow id does not belong to the pipeline",
		)
	}
	_, runner, err := mobilerunner.ResolveDevice(app, canonify.NormalizePath(deviceID))
	if err != nil ||
		mobilerunner.RunnerNamespace(runner.Id) != handlerNamespace {
		return nexus.NewHandlerErrorf(
			nexus.HandlerErrorTypeBadRequest,
			"device %s is not hosted by this runner",
			deviceID,
		)
	}
	return nil
}

// requirePipelineRunCaller accepts a caller link naming the pipeline run
// itself, or, for a step, a linked-steps block the pipeline run started:
// steps inside a block are scheduled by the block.
func requirePipelineRunCaller(
	ctx context.Context,
	describe organizationWorkflowDescriber,
	operation string,
	pipelineRun mobiledevice.PipelineRun,
	callerWorkflowID, callerRunID string,
) error {
	if callerWorkflowID == pipelineRun.WorkflowID && callerRunID == pipelineRun.RunID {
		return nil
	}
	if operation != mobiledevice.RunStepOperationName ||
		callerWorkflowID == "" || callerRunID == "" {
		return notOrganizationPipeline()
	}
	described, err := describe(ctx, pipelineRun.Namespace, callerWorkflowID, callerRunID)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return notOrganizationPipeline()
		}
		return nexus.NewHandlerErrorf(
			nexus.HandlerErrorTypeInternal,
			"check operation caller: %v",
			err,
		)
	}
	info := described.GetWorkflowExecutionInfo()
	parent := info.GetParentExecution()
	if info.GetType().GetName() != pipelineinternal.LinkedStepsWorkflowName ||
		info.GetExecution().GetRunId() != callerRunID ||
		parent.GetWorkflowId() != pipelineRun.WorkflowID ||
		parent.GetRunId() != pipelineRun.RunID {
		return notOrganizationPipeline()
	}
	return nil
}

func notOrganizationPipeline() error {
	return nexus.NewHandlerErrorf(
		nexus.HandlerErrorTypeBadRequest,
		"operation caller is not an organization pipeline",
	)
}
