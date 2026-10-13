// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/safehttp"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

func getOrgIDfromName(name string) (string, error) { //nolint
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	filter := fmt.Sprintf(`name="%s"`, name)

	record, err := app.FindFirstRecordByFilter("organizations", filter)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func jsonBody(data map[string]any) *bytes.Reader {
	b, _ := json.Marshal(data)
	return bytes.NewReader(b)
}

func TestReadSchemaFile(t *testing.T) {
	tempDir := t.TempDir()
	path := tempDir + "/schema.json"
	require.NoError(t, os.WriteFile(path, []byte(`{"type":"object"}`), 0o600))

	content, apiErr := readSchemaFile(path)
	require.Nil(t, apiErr)
	require.Contains(t, content, `"type":"object"`)

	_, apiErr = readSchemaFile(tempDir + "/missing.json")
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "failed to read  JSON schema file", apiErr.Reason)
}

func TestCheckWellKnownEndpoints(t *testing.T) {
	ctx := context.Background()

	err := checkWellKnownEndpoints(ctx, "http://127.0.0.1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "neither .well-known")

	err = checkWellKnownEndpoints(ctx, "http://127.0.0.1/.well-known/openid-federation")
	require.Error(t, err)
	require.Contains(t, err.Error(), "is not accessible")
}

func TestHandleCredentialIssuerStartCheckBadURL(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"::::://bad-url"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleCredentialIssuerStartCheckSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgID, err := pbutils.GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)

	issuerCollection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)

	issuerRecord := core.NewRecord(issuerCollection)
	issuerRecord.Set("url", "https://issuer.example.com")
	issuerRecord.Set("name", "Existing Issuer")
	issuerRecord.Set("owner", orgID)
	issuerRecord.Set("imported", true)
	require.NoError(t, app.Save(issuerRecord))

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	origClient := credentialIssuerTemporalClient
	origWait := credentialIssuerWaitForUpdateResult
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
		credentialIssuerTemporalClient = origClient
		credentialIssuerWaitForUpdateResult = origWait
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-issuer",
			WorkflowRunID: "run-issuer",
		}, nil
	}
	credentialIssuerTemporalClient = func(string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	credentialIssuerWaitForUpdateResult = func(
		context.Context,
		client.Client,
		string,
		string,
		string,
	) (map[string]any, error) {
		return map[string]any{
			"issuerName":        "Issuer Name",
			"logo":              "https://logo.example.com/logo.png",
			"credentialsNumber": 2.0,
		}, nil
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	require.Equal(t, float64(2), payload["credentialsNumber"])

	updated, err := app.FindRecordById("credential_issuers", issuerRecord.Id)
	require.NoError(t, err)
	require.NotEmpty(t, updated.GetString("workflow_url"))
}

func TestHandleCredentialIssuerStartCheckReadSchemaErrorAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) {
		return "", apierror.New(http.StatusBadRequest, "schema", "bad", "bad")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleCredentialIssuerStartCheckTemporalClientErrorAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	origClient := credentialIssuerTemporalClient
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
		credentialIssuerTemporalClient = origClient
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf", WorkflowRunID: "run"}, nil
	}
	credentialIssuerTemporalClient = func(string) (client.Client, error) {
		return nil, errors.New("no client")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.client.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleCredentialIssuerStartCheckUsesHostnameFallbackAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	origClient := credentialIssuerTemporalClient
	origWait := credentialIssuerWaitForUpdateResult
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
		credentialIssuerTemporalClient = origClient
		credentialIssuerWaitForUpdateResult = origWait
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf", WorkflowRunID: "run"}, nil
	}
	credentialIssuerTemporalClient = func(string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	credentialIssuerWaitForUpdateResult = func(
		context.Context,
		client.Client,
		string,
		string,
		string,
	) (map[string]any, error) {
		return map[string]any{
			"issuerName":        "",
			"logo":              "",
			"credentialsNumber": 1.0,
		}, nil
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.fallback.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	orgID, err := pbutils.GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)
	records, err := app.FindRecordsByFilter(
		"credential_issuers",
		"url = {:url} && owner = {:owner}",
		"",
		1,
		0,
		map[string]any{"url": "https://issuer.fallback.example.com", "owner": orgID},
	)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "issuer.fallback.example.com", records[0].GetString("name"))
}

func TestHandleCredentialIssuerStartCheckExistingRecordStartErrorAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgID, err := pbutils.GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)

	issuerCollection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)
	record := core.NewRecord(issuerCollection)
	record.Set("url", "https://issuer.existing.example.com")
	record.Set("name", "Existing")
	record.Set("owner", orgID)
	record.Set("imported", true)
	require.NoError(t, app.Save(record))

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("boom")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.existing.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	_, err = app.FindRecordById("credential_issuers", record.Id)
	require.NoError(t, err)
}

func TestHandleCredentialIssuerImportFidesSuccess(t *testing.T) {
	t.Setenv("ROOT_DIR", "../../../..")

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgID, err := pbutils.GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)
	orgName, err := pbutils.GetOrganizationCanonifiedName(app, orgID)
	require.NoError(t, err)

	origRead := credentialIssuerReadSchemaFile
	origStart := fidesCredentialIssuersStartWorkflow
	t.Cleanup(func() {
		credentialIssuerReadSchemaFile = origRead
		fidesCredentialIssuersStartWorkflow = origStart
	})

	var capturedNamespace string
	var capturedInput workflowengine.WorkflowInput
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) {
		return `{"type":"object"}`, nil
	}
	fidesCredentialIssuersStartWorkflow = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedInput = input
		return workflowengine.WorkflowResult{
			WorkflowID:    "fides-wf",
			WorkflowRunID: "fides-run",
		}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/credentials_issuers/import-fides", nil)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerImportFides()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	require.Equal(t, "fides-wf", payload["workflow_id"])
	require.Equal(t, "fides-run", payload["workflow_run_id"])
	require.Equal(t, orgName, capturedNamespace)
	require.Equal(t, orgID, capturedInput.Config["orgID"])
	require.Equal(t, `{"type":"object"}`, capturedInput.Config["issuer_schema"])
	require.NotEmpty(t, capturedInput.Config["app_url"])
}

func TestHandleCredentialIssuerImportFidesSchedule(t *testing.T) {
	t.Setenv("ROOT_DIR", "../../../..")

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://credimi.test"

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgID, err := pbutils.GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)
	orgName, err := pbutils.GetOrganizationCanonifiedName(app, orgID)
	require.NoError(t, err)

	origRead := credentialIssuerReadSchemaFile
	origStart := fidesCredentialIssuersStartWorkflow
	origTemporalClient := fidesCredentialIssuersTemporalClient
	t.Cleanup(func() {
		credentialIssuerReadSchemaFile = origRead
		fidesCredentialIssuersStartWorkflow = origStart
		fidesCredentialIssuersTemporalClient = origTemporalClient
	})

	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) {
		return `{"type":"object"}`, nil
	}
	fidesCredentialIssuersStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("immediate start should not be called")
	}

	mockHandle := &temporalmocks.ScheduleHandle{}
	mockHandle.On("Trigger", mock.Anything, fidesCredentialIssuersScheduleTriggerOptions).
		Return(nil).
		Once()
	mockClient := &temporalmocks.Client{}
	mockScheduleClient := &fakeScheduleClient{handle: mockHandle}
	mockClient.On("ScheduleClient").Return(mockScheduleClient)
	fidesCredentialIssuersTemporalClient = func(namespace string) (client.Client, error) {
		require.Equal(t, orgName, namespace)
		return mockClient, nil
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/import-fides",
		bytes.NewBufferString(`{"interval_days":3}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerImportFides()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	require.Equal(t, fidesCredentialIssuersScheduleID, payload["schedule_id"])
	require.Equal(t, orgName, payload["workflowNamespace"])

	require.Len(t, mockScheduleClient.createdOptions, 1)
	opts := mockScheduleClient.createdOptions[0]
	require.Equal(t, fidesCredentialIssuersScheduleID, opts.ID)
	require.Len(t, opts.Spec.Intervals, 1)
	require.Equal(t, 72*time.Hour, opts.Spec.Intervals[0].Every)

	action, ok := opts.Action.(*client.ScheduleWorkflowAction)
	require.True(t, ok)
	require.Equal(t, workflows.FidesCredentialIssuersWorkflowName, action.Workflow)
	require.Equal(t, workflows.FidesCredentialIssuersTaskQueue, action.TaskQueue)
	require.Len(t, action.Args, 1)
	workflowInput, ok := action.Args[0].(workflowengine.WorkflowInput)
	require.True(t, ok)
	require.Equal(t, orgID, workflowInput.Config["orgID"])
	require.Equal(t, `{"type":"object"}`, workflowInput.Config["issuer_schema"])
	require.Equal(t, "https://credimi.test", workflowInput.Config["app_url"])
	mockHandle.AssertExpectations(t)
}

func TestScheduleFidesCredentialIssuersImportUpdateKeepsState(t *testing.T) {
	origTemporalClient := fidesCredentialIssuersTemporalClient
	t.Cleanup(func() { fidesCredentialIssuersTemporalClient = origTemporalClient })

	var update client.ScheduleUpdateOptions
	mockHandle := &temporalmocks.ScheduleHandle{}
	mockHandle.On("Update", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			update = args.Get(1).(client.ScheduleUpdateOptions)
		}).
		Return(nil).
		Once()
	mockHandle.On("Trigger", mock.Anything, fidesCredentialIssuersScheduleTriggerOptions).
		Return(nil).
		Once()
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(&fakeScheduleClient{
		createErr: temporal.ErrScheduleAlreadyRunning,
		handle:    mockHandle,
	})
	fidesCredentialIssuersTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}

	_, err := scheduleFidesCredentialIssuersImport(
		context.Background(),
		"acme",
		workflowengine.WorkflowInput{},
		2,
	)
	require.NoError(t, err)
	mockHandle.AssertExpectations(t)

	state := &client.ScheduleState{Paused: true, Note: "paused by operator"}
	updated, err := update.DoUpdate(client.ScheduleUpdateInput{
		Description: client.ScheduleDescription{Schedule: client.Schedule{State: state}},
	})
	require.NoError(t, err)
	require.Same(t, state, updated.Schedule.State)
	require.Equal(t, 48*time.Hour, updated.Schedule.Spec.Intervals[0].Every)
}

func TestHandleCredentialIssuerStartCheckWorkflowErrorDeletesNewRecord(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, fmt.Errorf("boom")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.new.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	orgID, err := pbutils.GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)
	issuers, err := app.FindRecordsByFilter(
		"credential_issuers",
		"url = {:url} && owner = {:owner}",
		"",
		0,
		0,
		map[string]any{"url": "https://issuer.new.example.com", "owner": orgID},
	)
	require.NoError(t, err)
	require.Empty(t, issuers)
}

func TestHandleCredentialIssuerStartCheckWaitErrorDeletesNewRecord(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	origClient := credentialIssuerTemporalClient
	origWait := credentialIssuerWaitForUpdateResult
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
		credentialIssuerTemporalClient = origClient
		credentialIssuerWaitForUpdateResult = origWait
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-issuer",
			WorkflowRunID: "run-issuer",
		}, nil
	}
	credentialIssuerTemporalClient = func(string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	credentialIssuerWaitForUpdateResult = func(
		context.Context,
		client.Client,
		string,
		string,
		string,
	) (map[string]any, error) {
		return nil, fmt.Errorf("wait failed")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.wait.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	orgID, err := pbutils.GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)
	issuers, err := app.FindRecordsByFilter(
		"credential_issuers",
		"url = {:url} && owner = {:owner}",
		"",
		0,
		0,
		map[string]any{"url": "https://issuer.wait.example.com", "owner": orgID},
	)
	require.NoError(t, err)
	require.Empty(t, issuers)
}

func TestHandleCredentialIssuerStartCheckInvalidCredentialsNumber(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	origClient := credentialIssuerTemporalClient
	origWait := credentialIssuerWaitForUpdateResult
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
		credentialIssuerTemporalClient = origClient
		credentialIssuerWaitForUpdateResult = origWait
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-issuer",
			WorkflowRunID: "run-issuer",
		}, nil
	}
	credentialIssuerTemporalClient = func(string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	credentialIssuerWaitForUpdateResult = func(
		context.Context,
		client.Client,
		string,
		string,
		string,
	) (map[string]any, error) {
		return map[string]any{"issuerName": "Issuer", "credentialsNumber": "bad"}, nil
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.bad.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCheckEndpointExistsRefusesInternalDestinations(t *testing.T) {
	var internalHits atomic.Int32
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		internalHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer internal.Close()
	_, port, err := net.SplitHostPort(internal.Listener.Addr().String())
	require.NoError(t, err)
	wellKnown := "/.well-known/openid-credential-issuer"

	redirector := httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, internal.URL+wellKnown, http.StatusFound)
		}),
	)
	listener, err := net.Listen("tcp", "127.0.0.2:0")
	if err != nil {
		t.Skipf("127.0.0.2 not bindable: %v", err)
	}
	redirector.Listener = listener
	redirector.Start()
	defer redirector.Close()

	// Stand in for a public redirector: only its address may be dialed.
	origClient := credentialIssuerHTTPClient
	t.Cleanup(func() { credentialIssuerHTTPClient = origClient })
	redirectorIP := net.IPv4(127, 0, 0, 2)
	allowRedirector := func(ip net.IP) bool {
		return ip.Equal(redirectorIP) || safehttp.IsPublicIP(ip)
	}

	tests := []struct {
		name  string
		url   string
		allow func(net.IP) bool
	}{
		{name: "loopback", url: internal.URL + wellKnown},
		{name: "unspecified dials the local host", url: "http://0.0.0.0:" + port + wellKnown},
		{name: "localhost name", url: "http://localhost:" + port + wellKnown},
		{
			name:  "redirect to loopback",
			url:   redirector.URL + wellKnown,
			allow: allowRedirector,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			credentialIssuerHTTPClient = safehttp.NewClient(safehttp.Config{
				Timeout:      5 * time.Second,
				MaxRedirects: 10,
				Allow:        tc.allow,
			})
			err := checkEndpointExists(context.Background(), tc.url)
			require.ErrorIs(t, err, safehttp.ErrBlockedDestination)
		})
	}
	require.Zero(t, internalHits.Load(), "the internal server must never be contacted")

	credentialIssuerHTTPClient = safehttp.NewClient(safehttp.Config{
		Timeout:      5 * time.Second,
		MaxRedirects: 10,
		Allow:        func(net.IP) bool { return true },
	})
	require.NoError(t, checkEndpointExists(context.Background(), redirector.URL+wellKnown),
		"an allowed destination must still be reachable through a redirect")
	require.Equal(t, int32(1), internalHits.Load())
}

func TestCheckEndpointExistsInvalidURL(t *testing.T) {
	err := checkEndpointExists(context.Background(), "://bad")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid")
}

func TestCheckEndpointExistsUnsupportedScheme(t *testing.T) {
	err := checkEndpointExists(context.Background(), "ftp://example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported URL scheme")
}
