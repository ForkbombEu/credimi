// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestHandleMobileRunnerLifecycleLiveView(t *testing.T) {
	const orgNamespace = "usera-s-organization"
	pipelineType := pipeline.NewPipelineWorkflow().Name()
	describe := func(workflowType string, status enums.WorkflowExecutionStatus) func() (
		*workflowservice.DescribeWorkflowExecutionResponse, error,
	) {
		return func() (*workflowservice.DescribeWorkflowExecutionResponse, error) {
			return &workflowservice.DescribeWorkflowExecutionResponse{
				WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
					Type:   &common.WorkflowType{Name: workflowType},
					Status: status,
				},
			}, nil
		}
	}
	running := enums.WORKFLOW_EXECUTION_STATUS_RUNNING
	cases := []struct {
		name string
		// namespace is the queried namespace; "runner" stands for the runner's.
		namespace  string
		workflowID string
		describe   func() (*workflowservice.DescribeWorkflowExecutionResponse, error)
		want       bool
		wantStatus int
	}{
		{
			name:       "running pipeline that used the runner",
			namespace:  orgNamespace,
			workflowID: "pipeline-1",
			describe:   describe(pipelineType, running),
			want:       true,
		},
		{
			name:       "running pipeline without a result of this runner",
			namespace:  orgNamespace,
			workflowID: "pipeline-2",
			describe:   describe(pipelineType, running),
		},
		{
			name:       "closed pipeline",
			namespace:  orgNamespace,
			workflowID: "pipeline-1",
			describe:   describe(pipelineType, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED),
		},
		{
			name:       "other workflow type in an organization namespace",
			namespace:  orgNamespace,
			workflowID: "pipeline-1",
			describe:   describe(activities.ActionAuthorSessionWorkflowName, running),
		},
		{
			name:       "namespace of no organization",
			namespace:  "elsewhere",
			workflowID: "pipeline-1",
		},
		{
			name:       "running action author session in the runner namespace",
			namespace:  "runner",
			workflowID: "ActionAuthorSession-x-1",
			describe:   describe(activities.ActionAuthorSessionWorkflowName, running),
			want:       true,
		},
		{
			name:       "pipeline type in the runner namespace",
			namespace:  "runner",
			workflowID: "pipeline-1",
			describe:   describe(pipelineType, running),
		},
		{
			name:       "missing workflow",
			namespace:  orgNamespace,
			workflowID: "pipeline-1",
			describe: func() (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return nil, serviceerror.NewNotFound("missing")
			},
		},
		{
			name:       "temporal failure",
			namespace:  orgNamespace,
			workflowID: "pipeline-1",
			describe: func() (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return nil, errors.New("unavailable")
			},
			wantStatus: http.StatusServiceUnavailable,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := setupMobileRunnerApp(t)
			defer app.Cleanup()
			user, _ := setupMobileRunnerWorkTest(t, app)
			runner, err := canonify.Resolve(app, "/usera-s-organization/work-runner")
			require.NoError(t, err)
			orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
			require.NoError(t, err)
			device, err := app.FindFirstRecordByFilter(
				"mobile_devices",
				"runner = {:runner}",
				map[string]any{"runner": runner.Id},
			)
			require.NoError(t, err)
			savePipelineResult(t, app, orgID, "pipeline-1", "run-1", device.Id)

			namespace := tc.namespace
			if namespace == "runner" {
				namespace = mobilerunner.RunnerNamespace(runner.Id)
			}
			origClient := mobileRunnerLifecycleTemporalClient
			t.Cleanup(func() { mobileRunnerLifecycleTemporalClient = origClient })
			mockClient := temporalmocks.NewClient(t)
			if tc.describe != nil {
				resp, describeErr := tc.describe()
				mockClient.On("DescribeWorkflow", mock.Anything, tc.workflowID, "run-1").
					Return(temporalclient.Description(resp), describeErr).Once()
			}
			mobileRunnerLifecycleTemporalClient = func(ns string) (client.Client, error) {
				require.NotNil(t, tc.describe, "unexpected Temporal client")
				require.Equal(t, namespace, ns)
				return mockClient, nil
			}

			target := "/api/mobile-runner/lifecycle/live-view?" + url.Values{
				"runner_id":   {"/usera-s-organization/work-runner"},
				"namespace":   {namespace},
				"workflow_id": {tc.workflowID},
				"run_id":      {"run-1"},
			}.Encode()
			event := &core.RequestEvent{
				App:  app,
				Auth: user,
				Event: router.Event{
					Request:  httptest.NewRequest(http.MethodGet, target, nil),
					Response: httptest.NewRecorder(),
				},
			}
			err = HandleMobileRunnerLifecycleLiveView()(event)
			recorder := responseRecorder(t, event)
			requireHandlerErrorHandled(t, recorder, err)
			if tc.wantStatus != 0 {
				require.Equal(t, tc.wantStatus, recorder.Code, recorder.Body.String())
				return
			}
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			var response MobileRunnerLiveViewStatusResponse
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			require.Equal(t, tc.want, response.Running)
		})
	}
}

// savePipelineResult saves the pipeline_results row of a pipeline run of
// orgID that used deviceIDs.
func savePipelineResult(
	t *testing.T,
	app core.App,
	orgID, workflowID, runID string,
	deviceIDs ...string,
) {
	t.Helper()
	pipelines, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelines)
	pipelineRecord.Set("owner", orgID)
	pipelineRecord.Set("name", "live-view-pipeline")
	pipelineRecord.Set("canonified_name", "live-view-pipeline")
	pipelineRecord.Set("description", "demo pipeline")
	pipelineRecord.Set("yaml", "name: demo")
	require.NoError(t, app.Save(pipelineRecord))
	results, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	result := core.NewRecord(results)
	result.Set("owner", orgID)
	result.Set("pipeline", pipelineRecord.Id)
	result.Set("workflow_id", workflowID)
	result.Set("run_id", runID)
	result.Set("devices", deviceIDs)
	require.NoError(t, app.Save(result))
}
