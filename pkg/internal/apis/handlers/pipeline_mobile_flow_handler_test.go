// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/signedtoken"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
)

type pipelineMobileFlowEncodedValue struct {
	devices map[string]any
}

func (v pipelineMobileFlowEncodedValue) HasValue() bool {
	return true
}

func (v pipelineMobileFlowEncodedValue) Get(valuePtr interface{}) error {
	target := valuePtr.(*map[string]any)
	*target = v.devices
	return nil
}

func TestPipelineMobileFlowDevice(t *testing.T) {
	client := temporalmocks.NewClient(t)
	client.On(
		"DescribeWorkflow",
		mock.Anything,
		"pipeline-1",
		"run-1",
	).Return(temporalclient.Description(pipelineMobileFlowDescription(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, []string{
		"tenant/runner-1",
	})), nil).Once()
	client.On(
		"QueryWorkflow",
		mock.Anything,
		"pipeline-1",
		"run-1",
		pipeline.PipelineMobileDevicesQuery,
	).Return(converter.EncodedValue(pipelineMobileFlowEncodedValue{
		devices: map[string]any{
			"tenant/runner-1": map[string]any{
				"serial": "emulator-5554",
				"type":   "android_emulator",
			},
		},
	}), nil).Once()

	runnerID, device, apiErr := pipelineMobileFlowDevice(
		context.Background(),
		client,
		"pipeline-1",
		"run-1",
	)
	require.Nil(t, apiErr)
	require.Equal(t, "tenant/runner-1", runnerID)
	require.Equal(t, "emulator-5554", device["serial"])
}

func TestPipelineMobileFlowDeviceRejectsMultipleDevices(t *testing.T) {
	client := temporalmocks.NewClient(t)
	client.On(
		"DescribeWorkflow",
		mock.Anything,
		"pipeline-1",
		"run-1",
	).Return(temporalclient.Description(pipelineMobileFlowDescription(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, []string{
		"tenant/runner-1",
		"tenant/runner-2",
	})), nil).Once()

	_, _, apiErr := pipelineMobileFlowDevice(context.Background(), client, "pipeline-1", "run-1")
	require.NotNil(t, apiErr)
	require.Contains(t, apiErr.Error(), "pipeline must have exactly one reserved device")
}

func TestPipelineMobileFlowDeviceRequiresInitializedDevice(t *testing.T) {
	client := temporalmocks.NewClient(t)
	client.On(
		"DescribeWorkflow",
		mock.Anything,
		"pipeline-1",
		"run-1",
	).Return(temporalclient.Description(pipelineMobileFlowDescription(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, []string{
		"tenant/runner-1",
	})), nil).Once()
	client.On(
		"QueryWorkflow",
		mock.Anything,
		"pipeline-1",
		"run-1",
		pipeline.PipelineMobileDevicesQuery,
	).Return(converter.EncodedValue(pipelineMobileFlowEncodedValue{
		devices: map[string]any{},
	}), nil).Once()

	_, _, apiErr := pipelineMobileFlowDevice(context.Background(), client, "pipeline-1", "run-1")
	require.NotNil(t, apiErr)
	require.Contains(t, apiErr.Error(), "pipeline mobile device is not initialized")
}

func TestPipelineMobileFlowDeviceRequiresRunningPipeline(t *testing.T) {
	client := temporalmocks.NewClient(t)
	client.On(
		"DescribeWorkflow",
		mock.Anything,
		"pipeline-1",
		"run-1",
	).Return(temporalclient.Description(pipelineMobileFlowDescription(t, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, []string{
		"tenant/runner-1",
	})), nil).Once()

	_, _, apiErr := pipelineMobileFlowDevice(context.Background(), client, "pipeline-1", "run-1")
	require.NotNil(t, apiErr)
	require.Contains(t, apiErr.Error(), "pipeline workflow is not running")
}

func TestPipelineMobileFlowResponseIncludesOutput(t *testing.T) {
	response := pipelineMobileFlowResponse(workflowengine.WorkflowResult{
		Output: map[string]any{
			"status": "done",
		},
	}, nil)

	require.True(t, response.Success)
	require.Equal(t, map[string]any{"status": "done"}, response.Output)
	require.Nil(t, response.Error)
}

func TestPipelineMobileFlowResponseIncludesError(t *testing.T) {
	response := pipelineMobileFlowResponse(
		workflowengine.WorkflowResult{},
		errors.New("maestro flow failed"),
	)

	require.False(t, response.Success)
	require.Nil(t, response.Output)
	require.Equal(t, "maestro flow failed", response.Error)
}

func TestPipelineMobileFlowActivityOptions(t *testing.T) {
	options := pipelineMobileFlowActivityOptions()

	require.Equal(t, 20*time.Minute, options.ScheduleToCloseTimeout)
	require.Equal(t, 20*time.Minute, options.StartToCloseTimeout)
	require.NotNil(t, options.RetryPolicy)
	require.Equal(t, int32(1), options.RetryPolicy.MaximumAttempts)
}

func TestPipelineMobileFlowConfig(t *testing.T) {
	config := pipelineMobileFlowConfig("acme", "pipeline-1", "run-1", "acme/runner-1")

	require.Equal(t, map[string]any{
		"taskqueue":      fmt.Sprintf("%s-TaskQueue", canonify.NormalizePath("acme/runner-1")),
		"run_identifier": pipeline.RunIdentifier("acme", "pipeline-1", "run-1"),
	}, config)
}

func pipelineMobileFlowDescription(
	t *testing.T,
	status enums.WorkflowExecutionStatus,
	runnerIDs []string,
) *workflowservice.DescribeWorkflowExecutionResponse {
	t.Helper()

	runnerPayload, err := converter.GetDefaultDataConverter().ToPayload(runnerIDs)
	require.NoError(t, err)

	return &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
			Type:   &commonpb.WorkflowType{Name: "Dynamic Pipeline Workflow"},
			Status: status,
			SearchAttributes: &commonpb.SearchAttributes{
				IndexedFields: map[string]*commonpb.Payload{
					workflowengine.DeviceIdentifiersSearchAttribute: runnerPayload,
				},
			},
		},
	}
}

// A pipeline may run only its organization's actions and published ones.
func TestPipelineMobileFlowRefusesUnpublishedActionOfAnotherOrganization(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	other, err := app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", other.Id)
	wallet.Set("name", "foreign wallet")
	require.NoError(t, app.Save(wallet))
	actions, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	action := core.NewRecord(actions)
	action.Set("owner", other.Id)
	action.Set("wallet", wallet.Id)
	action.Set("name", "foreign action")
	action.Set("code", "appId: eu.wallet\n---\n- launchApp\n")
	action.Set("category", "onboarding")
	require.NoError(t, app.Save(action))
	actionPath := strings.Join([]string{
		other.GetString("canonified_name"),
		wallet.GetString("canonified_name"),
		action.GetString("canonified_name"),
	}, "/")
	resolved, err := canonify.Resolve(app, actionPath)
	require.NoError(t, err)
	require.Equal(t, action.Id, resolved.Id)

	mockClient := temporalmocks.NewClient(t)
	mockClient.On("DescribeWorkflow", mock.Anything, "pipeline-1", "run-1").
		Return(temporalclient.Description(pipelineMobileFlowDescription(
			t,
			enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
			[]string{"tenant/runner-1"},
		)), nil).Once()
	mockClient.On("QueryWorkflow", mock.Anything, "pipeline-1", "run-1", pipeline.PipelineMobileDevicesQuery).
		Return(converter.EncodedValue(pipelineMobileFlowEncodedValue{
			devices: map[string]any{"tenant/runner-1": map[string]any{
				"serial":           "emulator-5554",
				"type":             "android_emulator",
				"runner_namespace": "runner-abc",
			}},
		}), nil).
		Once()
	original := pipelineMobileFlowTemporalClient
	t.Cleanup(func() { pipelineMobileFlowTemporalClient = original })
	pipelineMobileFlowTemporalClient = func(namespace string) (client.Client, error) {
		require.Equal(t, "usera-s-organization", namespace, "no client for the runner namespace")
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/mobile-flow", nil)
	req = req.WithContext(context.WithValue(
		req.Context(),
		middlewares.ValidatedInputKey,
		PipelineMobileFlowInput{
			WorkflowID:      "pipeline-1",
			RunID:           "run-1",
			OrganizationID:  "usera-s-organization",
			ActionID:        actionPath,
			MobileFlowToken: mobileFlowToken(t, app, "usera-s-organization", "pipeline-1", "run-1"),
		},
	))
	rec := httptest.NewRecorder()
	err = HandlePipelineMobileFlow()(&core.RequestEvent{
		App:   app,
		Event: router.Event{Request: req, Response: rec},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	require.Equal(t, "wallet action not found", decodeAPIError(t, rec).Reason)
}

// mobileFlowToken signs the mobile flow token of a pipeline run.
func mobileFlowToken(t *testing.T, app core.App, namespace, workflowID, runID string) string {
	t.Helper()
	token, err := signedtoken.Sign(app, activities.MobileFlowTokenLabel, activities.MobileFlowGrant{
		Namespace:  namespace,
		WorkflowID: workflowID,
		RunID:      runID,
	})
	require.NoError(t, err)
	return token
}

func TestPipelineMobileFlowRequiresToken(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	body := `{"workflow_id":"pipeline-1","run_id":"run-1",` +
		`"organization_id":"usera-s-organization","action_id":"a/b/c"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/pipeline/mobile-flow",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	err := middlewares.DynamicValidateInputByType(reflect.TypeOf(PipelineMobileFlowInput{}))(
		&core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}},
	)
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "MobileFlowToken")
}

// A token opens only the pipeline run it was signed for, before any Temporal
// client is opened.
func TestPipelineMobileFlowRefusesTokenOfAnotherRun(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	original := pipelineMobileFlowTemporalClient
	t.Cleanup(func() { pipelineMobileFlowTemporalClient = original })
	pipelineMobileFlowTemporalClient = func(string) (client.Client, error) {
		t.Fatal("no Temporal client for a refused token")
		return nil, nil
	}
	for name, token := range map[string]string{
		"another run":          mobileFlowToken(t, app, "usera-s-organization", "pipeline-1", "run-2"),
		"another workflow":     mobileFlowToken(t, app, "usera-s-organization", "pipeline-2", "run-1"),
		"another organization": mobileFlowToken(t, app, "userb-s-organization", "pipeline-1", "run-1"),
		"forged":               "eyJ9.x",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/pipeline/mobile-flow", nil)
			req = req.WithContext(context.WithValue(
				req.Context(),
				middlewares.ValidatedInputKey,
				PipelineMobileFlowInput{
					WorkflowID:      "pipeline-1",
					RunID:           "run-1",
					OrganizationID:  "usera-s-organization",
					ActionID:        "a/b/c",
					MobileFlowToken: token,
				},
			))
			rec := httptest.NewRecorder()
			err := HandlePipelineMobileFlow()(&core.RequestEvent{
				App:   app,
				Event: router.Event{Request: req, Response: rec},
			})
			requireHandlerErrorHandled(t, rec, err)
			require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
			require.Equal(t, "invalid mobile flow token", decodeAPIError(t, rec).Reason)
		})
	}
}

// A setted_devices entry without runner_namespace was never prepared in its
// runner namespace.
func TestPipelineMobileFlowRefusesUnpreparedDevice(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	org, err := app.FindFirstRecordByData(
		"organizations",
		"canonified_name",
		"usera-s-organization",
	)
	require.NoError(t, err)
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", org.Id)
	wallet.Set("name", "own wallet")
	require.NoError(t, app.Save(wallet))
	actions, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	action := core.NewRecord(actions)
	action.Set("owner", org.Id)
	action.Set("wallet", wallet.Id)
	action.Set("name", "own action")
	action.Set("code", "appId: eu.wallet\n---\n- launchApp\n")
	action.Set("category", "onboarding")
	require.NoError(t, app.Save(action))
	actionPath := strings.Join([]string{
		org.GetString("canonified_name"),
		wallet.GetString("canonified_name"),
		action.GetString("canonified_name"),
	}, "/")

	mockClient := temporalmocks.NewClient(t)
	mockClient.On("DescribeWorkflow", mock.Anything, "pipeline-1", "run-1").
		Return(temporalclient.Description(pipelineMobileFlowDescription(
			t,
			enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
			[]string{"tenant/runner-1"},
		)), nil).Once()
	mockClient.On("QueryWorkflow", mock.Anything, "pipeline-1", "run-1", pipeline.PipelineMobileDevicesQuery).
		Return(converter.EncodedValue(pipelineMobileFlowEncodedValue{
			devices: map[string]any{"tenant/runner-1": map[string]any{
				"serial": "emulator-5554",
				"type":   "android_emulator",
			}},
		}), nil).
		Once()
	original := pipelineMobileFlowTemporalClient
	t.Cleanup(func() { pipelineMobileFlowTemporalClient = original })
	pipelineMobileFlowTemporalClient = func(namespace string) (client.Client, error) {
		require.Equal(t, "usera-s-organization", namespace, "no client for the runner namespace")
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/mobile-flow", nil)
	req = req.WithContext(context.WithValue(
		req.Context(),
		middlewares.ValidatedInputKey,
		PipelineMobileFlowInput{
			WorkflowID:      "pipeline-1",
			RunID:           "run-1",
			OrganizationID:  "usera-s-organization",
			ActionID:        actionPath,
			MobileFlowToken: mobileFlowToken(t, app, "usera-s-organization", "pipeline-1", "run-1"),
		},
	))
	rec := httptest.NewRecorder()
	err = HandlePipelineMobileFlow()(&core.RequestEvent{
		App:   app,
		Event: router.Event{Request: req, Response: rec},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	require.Equal(t, "device is not prepared", decodeAPIError(t, rec).Reason)
}
