// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
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

// callerLink is the link a pipeline in namespace adds to the operations it
// starts.
func callerLink(namespace string) nexus.Link {
	return temporalnexus.ConvertLinkWorkflowEventToNexusLink(&commonpb.Link_WorkflowEvent{
		Namespace:  namespace,
		WorkflowId: "pipeline-1",
		RunId:      "run-1",
		Reference: &commonpb.Link_WorkflowEvent_EventRef{
			EventRef: &commonpb.Link_WorkflowEvent_EventReference{
				EventId:   5,
				EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
			},
		},
	})
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
	cases := []struct {
		name        string
		links       []nexus.Link
		pipelineRun mobiledevice.PipelineRun
		workflowID  string
		deviceID    string
		wantErr     string
	}{
		{
			name:        "accepts an organization pipeline",
			links:       []nexus.Link{callerLink(orgNamespace)},
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    deviceID,
		},
		{
			name:        "rejects a caller without link",
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    deviceID,
			wantErr:     "operation caller is not an organization pipeline",
		},
		{
			name:  "rejects a runner namespace caller",
			links: []nexus.Link{callerLink(handlerNamespace)},
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
			links:       []nexus.Link{callerLink("another-org")},
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    deviceID,
			wantErr:     "operation caller is not an organization pipeline",
		},
		{
			name:  "rejects a namespace of no organization",
			links: []nexus.Link{callerLink("not-an-org")},
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
			name:        "rejects a workflow id of another pipeline",
			links:       []nexus.Link{callerLink(orgNamespace)},
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-2-step-1",
			deviceID:    deviceID,
			wantErr:     "workflow id does not belong to the pipeline",
		},
		{
			name:        "rejects the pipeline's own workflow id",
			links:       []nexus.Link{callerLink(orgNamespace)},
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1",
			deviceID:    deviceID,
			wantErr:     "workflow id does not belong to the pipeline",
		},
		{
			name:        "rejects a device of another runner",
			links:       []nexus.Link{callerLink(orgNamespace)},
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    foreignDeviceID,
			wantErr:     "device " + foreignDeviceID + " is not hosted by this runner",
		},
		{
			name:        "rejects an unknown device",
			links:       []nexus.Link{callerLink(orgNamespace)},
			pipelineRun: pipelineRun,
			workflowID:  "pipeline-1-step-1",
			deviceID:    "missing/device",
			wantErr:     "device missing/device is not hosted by this runner",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := requireCredimiCaller(
				app,
				nexus.StartOperationOptions{Links: tt.links},
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
