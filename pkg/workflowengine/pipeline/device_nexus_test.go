// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/temporalnexus"
)

// saveCallerFixture saves a record of collection with fields.
func saveCallerFixture(
	t *testing.T,
	app *tests.TestApp,
	collection string,
	fields map[string]any,
) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId(collection)
	require.NoError(t, err)
	record := core.NewRecord(coll)
	for key, value := range fields {
		record.Set(key, value)
	}
	require.NoError(t, app.Save(record))
	return record
}

// callerLink is the link the workflow run workflowID/runID of namespace adds
// to the operations it starts.
func callerLink(namespace, workflowID, runID string) nexus.Link {
	return temporalnexus.ConvertLinkWorkflowEventToNexusLink(&commonpb.Link_WorkflowEvent{
		Namespace:  namespace,
		WorkflowId: workflowID,
		RunId:      runID,
		Reference: &commonpb.Link_WorkflowEvent_EventRef{
			EventRef: &commonpb.Link_WorkflowEvent_EventReference{
				EventId:   5,
				EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
			},
		},
	})
}

// fakeDescriber describes the workflows of runs, keyed by workflow ID and
// run ID, and answers NotFound for any other.
func fakeDescriber(
	runs map[[2]string]*workflowpb.WorkflowExecutionInfo,
) organizationWorkflowDescriber {
	return func(
		_ context.Context,
		_, workflowID, runID string,
	) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
		info, ok := runs[[2]string{workflowID, runID}]
		if !ok {
			return nil, serviceerror.NewNotFound("workflow not found")
		}
		return &workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: info,
		}, nil
	}
}

func TestRequireCredimiCaller(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	canonify.RegisterCanonifyHooks(app)

	org, err := app.FindFirstRecordByFilter("organizations", "1=1")
	require.NoError(t, err)
	orgNamespace := org.GetString("canonified_name")
	require.NotEmpty(t, orgNamespace)

	newRunnerDevice := func(name string) (runner *core.Record, deviceID string) {
		runner = saveCallerFixture(t, app, "mobile_runners", map[string]any{
			"owner": org.Id,
			"name":  name,
			"ip":    "http://runner.test",
			"type":  "android_emulator",
		})
		device := saveCallerFixture(t, app, "mobile_devices", map[string]any{
			"owner":  org.Id,
			"runner": runner.Id,
			"name":   name + "-device",
			"type":   "android_emulator",
			"serial": name + "-serial",
		})
		path, err := canonify.BuildPath(
			app,
			device,
			canonify.CanonifyPaths["mobile_devices"],
			"",
		)
		require.NoError(t, err)
		return runner, canonify.NormalizePath(path)
	}
	runner, deviceID := newRunnerDevice("caller-runner")
	_, foreignDeviceID := newRunnerDevice("other-runner")
	handlerNamespace := mobilerunner.RunnerNamespace(runner.Id)

	pipelineRun := mobiledevice.PipelineRun{
		Namespace:  orgNamespace,
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
	}
	longPipelineRun := mobiledevice.PipelineRun{
		Namespace:  orgNamespace,
		WorkflowID: strings.Repeat("p", 300),
		RunID:      "run-1",
	}
	pipelineLink := callerLink(orgNamespace, "pipeline-1", "run-1")
	blockLink := callerLink(orgNamespace, "pipeline-1-block", "block-run-1")
	otherBlockLink := callerLink(orgNamespace, "pipeline-2-block", "block-run-2")
	runs := map[[2]string]*workflowpb.WorkflowExecutionInfo{
		{"pipeline-1-block", "block-run-1"}: {
			Execution: &commonpb.WorkflowExecution{
				WorkflowId: "pipeline-1-block",
				RunId:      "block-run-1",
			},
			Type: &commonpb.WorkflowType{Name: pipelineinternal.LinkedStepsWorkflowName},
			ParentExecution: &commonpb.WorkflowExecution{
				WorkflowId: "pipeline-1",
				RunId:      "run-1",
			},
		},
		{"pipeline-2-block", "block-run-2"}: {
			Execution: &commonpb.WorkflowExecution{
				WorkflowId: "pipeline-2-block",
				RunId:      "block-run-2",
			},
			Type: &commonpb.WorkflowType{Name: pipelineinternal.LinkedStepsWorkflowName},
			ParentExecution: &commonpb.WorkflowExecution{
				WorkflowId: "pipeline-2",
				RunId:      "run-2",
			},
		},
	}
	prepare := mobiledevice.PrepareOperationName
	runStep := mobiledevice.RunStepOperationName
	cleanUp := mobiledevice.CleanUpOperationName
	prepareID := mobiledevice.PrepareWorkflowID("pipeline-1", deviceID)
	cases := []struct {
		name        string
		links       []nexus.Link
		operation   string
		pipelineRun mobiledevice.PipelineRun
		workflowID  string
		deviceID    string
		wantErr     string
	}{
		{
			name:        "accepts a step of the pipeline run",
			links:       []nexus.Link{pipelineLink},
			operation:   runStep,
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    deviceID,
		},
		{
			name:        "accepts the prepare of the pipeline run",
			links:       []nexus.Link{pipelineLink},
			operation:   prepare,
			pipelineRun: pipelineRun,
			workflowID:  prepareID,
			deviceID:    deviceID,
		},
		{
			name:        "accepts the cleanup of the pipeline run",
			links:       []nexus.Link{pipelineLink},
			operation:   cleanUp,
			pipelineRun: pipelineRun,
			workflowID:  mobiledevice.CleanUpWorkflowID("pipeline-1", deviceID),
			deviceID:    deviceID,
		},
		{
			name: "accepts the prepare of a 300-byte pipeline ID",
			links: []nexus.Link{
				callerLink(orgNamespace, longPipelineRun.WorkflowID, "run-1"),
			},
			operation:   prepare,
			pipelineRun: longPipelineRun,
			workflowID:  mobiledevice.PrepareWorkflowID(longPipelineRun.WorkflowID, deviceID),
			deviceID:    deviceID,
		},
		{
			name:        "accepts a step of a linked-steps block of the run",
			links:       []nexus.Link{blockLink},
			operation:   runStep,
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-block-step-1",
			deviceID:    deviceID,
		},
		{
			name:        "rejects the prepare from a linked-steps block",
			links:       []nexus.Link{blockLink},
			operation:   prepare,
			pipelineRun: pipelineRun,
			workflowID:  prepareID,
			deviceID:    deviceID,
			wantErr:     "operation caller is not an organization pipeline",
		},
		{
			name:        "rejects a step from a block of another run",
			links:       []nexus.Link{otherBlockLink},
			operation:   runStep,
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    deviceID,
			wantErr:     "operation caller is not an organization pipeline",
		},
		{
			name:        "rejects a step from an unknown workflow",
			links:       []nexus.Link{callerLink(orgNamespace, "forged", "run-9")},
			operation:   runStep,
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    deviceID,
			wantErr:     "operation caller is not an organization pipeline",
		},
		{
			name:        "rejects a link to another run of the same ID",
			links:       []nexus.Link{callerLink(orgNamespace, "pipeline-1", "run-2")},
			operation:   prepare,
			pipelineRun: pipelineRun,
			workflowID:  prepareID,
			deviceID:    deviceID,
			wantErr:     "operation caller is not an organization pipeline",
		},
		{
			name:        "rejects a caller without link",
			operation:   runStep,
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    deviceID,
			wantErr:     "operation caller is not an organization pipeline",
		},
		{
			name:      "rejects a runner namespace caller",
			links:     []nexus.Link{callerLink(handlerNamespace, "pipeline-1", "run-1")},
			operation: runStep,
			pipelineRun: mobiledevice.PipelineRun{
				Namespace:  handlerNamespace,
				WorkflowID: "pipeline-1",
				RunID:      "run-1",
			},
			workflowID: "pipeline-1-step-1",
			deviceID:   deviceID,
			wantErr:    "operation caller is not an organization pipeline",
		},
		{
			name:        "rejects a link to another namespace than the pipeline's",
			links:       []nexus.Link{callerLink("another-org", "pipeline-1", "run-1")},
			operation:   runStep,
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    deviceID,
			wantErr:     "operation caller is not an organization pipeline",
		},
		{
			name:      "rejects a namespace of no organization",
			links:     []nexus.Link{callerLink("not-an-org", "pipeline-1", "run-1")},
			operation: runStep,
			pipelineRun: mobiledevice.PipelineRun{
				Namespace:  "not-an-org",
				WorkflowID: "pipeline-1",
				RunID:      "run-1",
			},
			workflowID: "pipeline-1-step-1",
			deviceID:   deviceID,
			wantErr:    "operation caller is not an organization pipeline",
		},
		{
			name:        "rejects a prepare with another workflow id",
			links:       []nexus.Link{pipelineLink},
			operation:   prepare,
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-device-other-prepare",
			deviceID:    deviceID,
			wantErr:     "workflow id does not belong to the pipeline",
		},
		{
			name:        "rejects a cleanup with the prepare workflow id",
			links:       []nexus.Link{pipelineLink},
			operation:   cleanUp,
			pipelineRun: pipelineRun,
			workflowID:  prepareID,
			deviceID:    deviceID,
			wantErr:     "workflow id does not belong to the pipeline",
		},
		{
			name:        "rejects a device of another runner",
			links:       []nexus.Link{pipelineLink},
			operation:   runStep,
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    foreignDeviceID,
			wantErr:     "device " + foreignDeviceID + " is not hosted by this runner",
		},
		{
			name:        "rejects an unknown device",
			links:       []nexus.Link{pipelineLink},
			operation:   runStep,
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    "missing/device",
			wantErr:     "device missing/device is not hosted by this runner",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := requireCredimiCaller(
				context.Background(),
				app,
				fakeDescriber(runs),
				nexus.StartOperationOptions{Links: tt.links},
				tt.operation,
				handlerNamespace,
				tt.pipelineRun,
				tt.workflowID,
				tt.deviceID,
			)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			var handlerErr *nexus.HandlerError
			require.True(t, errors.As(err, &handlerErr), "got %v", err)
			assert.Equal(t, nexus.HandlerErrorTypeBadRequest, handlerErr.Type)
			assert.Equal(t, tt.wantErr, handlerErr.Message)
		})
	}

	t.Run("retries when the caller cannot be described", func(t *testing.T) {
		err := requireCredimiCaller(
			context.Background(),
			app,
			func(
				context.Context,
				string, string, string,
			) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return nil, errors.New("temporal down")
			},
			nexus.StartOperationOptions{Links: []nexus.Link{blockLink}},
			runStep,
			handlerNamespace,
			pipelineRun,
			"pipeline-1-step-1",
			deviceID,
		)
		var handlerErr *nexus.HandlerError
		require.True(t, errors.As(err, &handlerErr), "got %v", err)
		assert.Equal(t, nexus.HandlerErrorTypeInternal, handlerErr.Type)
	})
}

func TestDeviceWorkflowOptionsKeepPipelineRunMemo(t *testing.T) {
	pipelineRun := mobiledevice.PipelineRun{
		Namespace:  "tenant",
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
	}
	options := deviceWorkflowOptions(
		"pipeline-1-step-1",
		time.Hour,
		pipelineRun,
		map[string]any{
			"test":                          "step memo",
			mobiledevice.PipelineRunMemoKey: "forged",
		},
		"step-1\n",
	)
	assert.Equal(t, "pipeline-1-step-1", options.ID)
	assert.Equal(t, PipelineTaskQueue, options.TaskQueue)
	assert.Equal(t, time.Hour, options.WorkflowExecutionTimeout)
	assert.Equal(t, "step-1", options.StaticSummary)
	assert.Equal(t, map[string]any{
		"test":                          "step memo",
		mobiledevice.PipelineRunMemoKey: pipelineRun,
	}, options.Memo)
}
