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
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
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
		wantReason string
		// held, for an action author session, is what its device query
		// answers: "device" stands for a device of the runner.
		held     map[string]any
		queryErr error
		// prepare changes the fixture before the request.
		prepare func(t *testing.T, app *tests.TestApp, user *core.Record)
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
			name:       "action author session holding a device of the runner",
			namespace:  "runner",
			workflowID: "ActionAuthorSession-x-1",
			describe:   describe(activities.ActionAuthorSessionWorkflowName, running),
			held:       map[string]any{"device": map[string]any{"type": "android_emulator"}},
			want:       true,
		},
		{
			name:       "action author session that released its device",
			namespace:  "runner",
			workflowID: "ActionAuthorSession-x-1",
			describe:   describe(activities.ActionAuthorSessionWorkflowName, running),
			held:       map[string]any{},
		},
		{
			name:       "action author session holding a device Credimi does not know",
			namespace:  "runner",
			workflowID: "ActionAuthorSession-x-1",
			describe:   describe(activities.ActionAuthorSessionWorkflowName, running),
			held: map[string]any{
				"usera-s-organization/other-runner/device": map[string]any{},
			},
		},
		{
			name:       "action author session whose query fails",
			namespace:  "runner",
			workflowID: "ActionAuthorSession-x-1",
			describe:   describe(activities.ActionAuthorSessionWorkflowName, running),
			queryErr:   errors.New("unavailable"),
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "action author session whose run the query does not find",
			namespace:  "runner",
			workflowID: "ActionAuthorSession-x-1",
			describe:   describe(activities.ActionAuthorSessionWorkflowName, running),
			queryErr:   serviceerror.NewNotFound("missing"),
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
		{
			name:       "member of the runner's organization",
			namespace:  orgNamespace,
			workflowID: "pipeline-1",
			prepare: func(t *testing.T, app *tests.TestApp, user *core.Record) {
				demoteToMember(t, app, user.Email())
			},
			wantStatus: http.StatusForbidden,
			wantReason: "runner_owner_role_required",
		},
		{
			name:       "owner of an admin-managed runner's organization",
			namespace:  orgNamespace,
			workflowID: "pipeline-1",
			prepare: func(t *testing.T, app *tests.TestApp, _ *core.Record) {
				setMobileRunnerAdminManaged(t, app, "usera-s-organization/work-runner", true)
			},
			wantStatus: http.StatusForbidden,
			wantReason: "admin_managed_runner",
		},
		{
			name:       "disabled runner",
			namespace:  orgNamespace,
			workflowID: "pipeline-1",
			prepare: func(t *testing.T, app *tests.TestApp, _ *core.Record) {
				runner, err := canonify.Resolve(app, "/usera-s-organization/work-runner")
				require.NoError(t, err)
				runner.Set("disabled", true)
				require.NoError(t, app.Save(runner))
			},
			wantStatus: http.StatusForbidden,
			wantReason: "runner_disabled",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := setupMobileRunnerApp(t)
			defer app.Cleanup()
			user, devicePaths := setupMobileRunnerWorkTest(t, app)
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
			if tc.prepare != nil {
				tc.prepare(t, app, user)
			}

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
			if tc.held != nil || tc.queryErr != nil {
				held := map[string]any{}
				for id, value := range tc.held {
					if id == "device" {
						id = devicePaths[0]
					}
					held[id] = value
				}
				var answer converter.EncodedValue
				if tc.queryErr == nil {
					answer = pipelineMobileFlowEncodedValue{devices: held}
				}
				mockClient.On(
					"QueryWorkflow",
					mock.Anything,
					tc.workflowID,
					"run-1",
					pipeline.PipelineMobileDevicesQuery,
				).Return(answer, tc.queryErr).Once()
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
				if tc.wantReason != "" {
					require.Equal(t, tc.wantReason, decodeAPIError(t, recorder).Reason)
				}
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
