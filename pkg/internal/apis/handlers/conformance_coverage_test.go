// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

var errCovAPIStart = errors.New("temporal unavailable")

func covAPIWriteTemplate(t *testing.T, rootDir, rel, content string) {
	t.Helper()
	path := filepath.Join(rootDir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func covAPIFailStarter(
	t *testing.T,
	target *func(workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error),
) {
	t.Helper()
	orig := *target
	t.Cleanup(func() { *target = orig })
	*target = func(workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errCovAPIStart
	}
}

func covAPIRequireStarterError(t *testing.T, err error, reason string) {
	t.Helper()
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	if reason != "" {
		assert.Equal(t, reason, apiErr.Reason)
	}
}

func TestConformanceStartersRejectInvalidInput(t *testing.T) {
	app := newStarterTestApp(t)

	walletV1 := workflows.OpenID4VPWalletStepCITemplatePathv1_0
	issuer := workflows.OpenID4VCIIssuerStepCITemplatePath
	ewcTemplate := filepath.Join(workflows.EWCTemplateFolderPath, "test.yaml")
	webuildTemplate := filepath.Join(workflows.WebuildTemplateFolderPath, "test.yaml")
	eudiwTemplate := filepath.Join(workflows.EudiwTemplateFolderPath, "test.yaml")

	scenarios := []struct {
		name     string
		template string
		start    WorkflowStarter
		params   WorkflowStarterParams
		failWith *func(workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error)
		reason   string
	}{
		{
			name:   "wallet requires yaml",
			start:  startOpenID4VPWalletWorkflow,
			params: WorkflowStarterParams{Version: "1.0"},
		},
		{
			name:   "wallet rejects malformed yaml",
			start:  startOpenID4VPWalletWorkflow,
			params: WorkflowStarterParams{YAMLData: "variant: [", Version: "1.0"},
			reason: "failed to parse YAML input",
		},
		{
			name:   "wallet rejects mistyped test name",
			start:  startOpenID4VPWalletWorkflow,
			params: WorkflowStarterParams{YAMLData: "variant: json\ntest: [1]\n", Version: "1.0"},
			reason: "failed to parse JSON input",
		},
		{
			name:  "wallet draft-24 template missing",
			start: startOpenID4VPWalletWorkflow,
			params: WorkflowStarterParams{
				YAMLData: "variant: json\ntest: t\n",
				Version:  "draft-24",
			},
			reason: "failed to read template file",
		},
		{
			name:     "wallet start failure",
			template: walletV1,
			start:    startOpenID4VPWalletWorkflow,
			params: WorkflowStarterParams{
				App:      app,
				YAMLData: "variant: json\ntest: t\n",
				Version:  "1.0",
			},
			failWith: &openID4VPWalletWorkflowStart,
			reason:   "failed to start workflow",
		},
		{
			name:   "issuer requires yaml",
			start:  startOpenID4VPWalletWorkflow,
			params: WorkflowStarterParams{Protocol: "openid4vci_issuer"},
			reason: "YAML data is required for OID4VCI issuer workflow",
		},
		{
			name:   "issuer template missing",
			start:  startOpenID4VPWalletWorkflow,
			params: WorkflowStarterParams{Protocol: "openid4vci_issuer", YAMLData: "test: t\n"},
			reason: "failed to read template file",
		},
		{
			name:     "issuer rejects malformed yaml",
			template: issuer,
			start:    startOpenID4VPWalletWorkflow,
			params:   WorkflowStarterParams{Protocol: "openid4vci_issuer", YAMLData: "test: ["},
			reason:   "failed to parse YAML input for OID4VCI issuer workflow",
		},
		{
			name:     "issuer start failure",
			template: issuer,
			start:    startOpenID4VPWalletWorkflow,
			params: WorkflowStarterParams{
				App:      app,
				Protocol: "openid4vci_issuer",
				YAMLData: "test: t\n",
			},
			failWith: &openID4VCIIssuerWorkflowStart,
			reason:   "failed to start OID4VCI issuer workflow",
		},
		{
			name:   "ewc template missing",
			start:  startEWCWorkflow,
			params: WorkflowStarterParams{TestName: "ewcmissing.yaml", YAMLData: "a: b\n"},
			reason: "failed to read template file",
		},
		{
			name:     "ewc rejects malformed yaml",
			template: ewcTemplate,
			start:    startEWCWorkflow,
			params:   WorkflowStarterParams{TestName: "ewctest.yaml", YAMLData: "a: ["},
			reason:   "failed to parse YAML input",
		},
		{
			name:     "webuild start failure",
			template: webuildTemplate,
			start:    startWebuildWorkflow,
			params: WorkflowStarterParams{
				App:      app,
				TestName: "webuildtest.yaml",
				YAMLData: "a: b\n",
				Protocol: workflows.OpenID4VPWalletStandard,
			},
			failWith: &webuildWorkflowStart,
			reason:   "failed to start workflow",
		},
		{
			name:   "eudiw template missing",
			start:  startEudiwWorkflow,
			params: WorkflowStarterParams{TestName: "eudiwmissing.yaml", YAMLData: "nonce: n\n"},
			reason: "failed to read template file",
		},
		{
			name:     "eudiw rejects malformed yaml",
			template: eudiwTemplate,
			start:    startEudiwWorkflow,
			params:   WorkflowStarterParams{TestName: "eudiwtest.yaml", YAMLData: "nonce: ["},
			reason:   "failed to parse YAML input",
		},
		{
			name:     "eudiw start failure",
			template: eudiwTemplate,
			start:    startEudiwWorkflow,
			params: WorkflowStarterParams{
				App:      app,
				TestName: "eudiwtest.yaml",
				YAMLData: "nonce: n\nid: i\n",
			},
			failWith: &eudiwWorkflowStart,
			reason:   "failed to start workflow",
		},
		{
			name:   "vlei requires yaml",
			start:  startvLEIWorkflow,
			params: WorkflowStarterParams{},
		},
		{
			name:   "vlei rejects malformed yaml",
			start:  startvLEIWorkflow,
			params: WorkflowStarterParams{YAMLData: "credentialID: ["},
			reason: "failed to parse YAML input",
		},
		{
			name:   "vlei rejects mistyped credential id",
			start:  startvLEIWorkflow,
			params: WorkflowStarterParams{YAMLData: "credentialID: [1]\n"},
			reason: "failed to parse JSON input",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			rootDir := t.TempDir()
			if s.template != "" {
				covAPIWriteTemplate(t, rootDir, s.template, "template")
			}
			t.Setenv("ROOT_DIR", rootDir)
			if s.failWith != nil {
				covAPIFailStarter(t, s.failWith)
			}

			_, err := s.start(s.params)
			covAPIRequireStarterError(t, err, s.reason)
			if s.failWith != nil {
				assert.ErrorContains(t, err, errCovAPIStart.Error())
			}
		})
	}
}

func TestStartvLEIWorkflowStartFailure(t *testing.T) {
	orig := vleiWorkflowStart
	t.Cleanup(func() { vleiWorkflowStart = orig })
	var namespace string
	vleiWorkflowStart = func(ns string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		namespace = ns
		return workflowengine.WorkflowResult{}, errCovAPIStart
	}

	_, err := startvLEIWorkflow(WorkflowStarterParams{
		App:       newStarterTestApp(t),
		YAMLData:  "credentialID: cred-1\n",
		Namespace: "org-ns",
	})
	covAPIRequireStarterError(t, err, "failed to start workflow")
	assert.Equal(t, "org-ns", namespace)
}

func TestStartOpenIDAutomatedConformanceWorkflowSplitsTestName(t *testing.T) {
	rootDir := t.TempDir()
	covAPIWriteTemplate(t, rootDir, workflows.OpenID4VCIIssuerStepCITemplatePath, "issuer template")
	t.Setenv("ROOT_DIR", rootDir)

	scenarios := []struct {
		name         string
		yaml         string
		wantTestName string
	}{
		{
			name:         "string test name",
			yaml:         "test: issuer-test\nissuer: https://i.test\n",
			wantTestName: "issuer-test",
		},
		{
			name:         "non-string test name is dropped",
			yaml:         "test: 5\nissuer: https://i.test\n",
			wantTestName: "",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			orig := openID4VCIIssuerWorkflowStart
			t.Cleanup(func() { openID4VCIIssuerWorkflowStart = orig })
			var started workflowengine.WorkflowInput
			openID4VCIIssuerWorkflowStart = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
				started = input
				return workflowengine.WorkflowResult{WorkflowID: "wf-issuer"}, nil
			}

			result, err := startOpenID4VPWalletWorkflow(WorkflowStarterParams{
				App:       newStarterTestApp(t),
				YAMLData:  s.yaml,
				Namespace: "ns",
				Author:    Author(workflows.OpenIDConformanceSuite),
				Protocol:  "openid4vci_issuer",
				UserName:  "User",
			})
			require.NoError(t, err)
			assert.Equal(t, workflows.OpenIDConformanceSuite, result.Author)

			payload, ok := started.Payload.(workflows.OpenIDConformanceWorkflowPayload)
			require.True(t, ok)
			assert.Equal(t, s.wantTestName, payload.TestName)
			assert.Equal(t, map[string]any{"issuer": "https://i.test"}, payload.Parameters)
			assert.Equal(t, "issuer template", started.Config["template"])
			assert.Equal(t, "https://app.example.com", started.Config["app_url"])
		})
	}
}

func covAPISaveAndStart(
	t *testing.T,
	app core.App,
	auth *core.Record,
	protocol, version string,
	input SaveVariablesAndStartRequestInput,
) (*httptest.ResponseRecorder, error) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/compliance/save-variables-and-start", nil)
	req.SetPathValue("protocol", protocol)
	req.SetPathValue("version", version)
	req = withValidatedInput(req, input)
	rec := httptest.NewRecorder()
	err := HandleSaveVariablesAndStart()(&core.RequestEvent{
		App:   app,
		Auth:  auth,
		Event: router.Event{Request: req, Response: rec},
	})
	return rec, err
}

func TestHandleSaveVariablesAndStartRejectsBadRequests(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	rootDir := t.TempDir()
	covAPIWriteTemplate(t, rootDir, "config_templates/ewc/v1/ewc/test-1.yaml", "foo: {{ .foo }}\n")
	covAPIWriteTemplate(t, rootDir, "config_templates/ewc/v1/unknown/test-1.yaml", "foo: bar\n")
	covAPIWriteTemplate(t, rootDir, "config_templates/ewc/v1/ewc/broken.yaml", "foo: {{ .foo \n")
	t.Setenv("ROOT_DIR", rootDir)

	origRegistry := workflowRegistry
	t.Cleanup(func() { workflowRegistry = origRegistry })
	workflowRegistry = map[Author]WorkflowStarter{
		"ewc": func(WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
			return workflowengine.WorkflowResult{}, errCovAPIStart
		},
	}

	variables := []Variable{{FieldName: "foo", Value: "bar", CredimiID: "cred-1"}}
	scenarios := []struct {
		name     string
		protocol string
		version  string
		input    SaveVariablesAndStartRequestInput
		reason   string
	}{
		{
			name:     "protocol escaping the templates dir",
			protocol: "..",
			version:  "..",
			reason:   "invalid protocol or version",
		},
		{
			name:     "unknown protocol directory",
			protocol: "ewc",
			version:  "v9",
			reason:   "directory does not exist for test ewc/v9",
		},
		{
			name:     "json check without author",
			protocol: "ewc",
			version:  "v1",
			input: SaveVariablesAndStartRequestInput{
				ConfigsWithJSON: map[string]string{"/t": "{}"},
			},
			reason: "author is required",
		},
		{
			name:     "json check start failure",
			protocol: "ewc",
			version:  "v1",
			input: SaveVariablesAndStartRequestInput{
				ConfigsWithJSON: map[string]string{"ewc/t": "{}"},
			},
			reason: "failed to process JSON checks",
		},
		{
			name:     "variables check without author",
			protocol: "ewc",
			version:  "v1",
			input: SaveVariablesAndStartRequestInput{
				ConfigsWithFields: map[string][]Variable{"/test-1.yaml": variables},
			},
			reason: "author is required",
		},
		{
			name:     "variables check with unsupported author",
			protocol: "ewc",
			version:  "v1",
			input: SaveVariablesAndStartRequestInput{
				ConfigsWithFields: map[string][]Variable{"unknown/test-1.yaml": variables},
			},
			reason: "failed to process variables test",
		},
		{
			name:     "variables template does not render",
			protocol: "ewc",
			version:  "v1",
			input: SaveVariablesAndStartRequestInput{
				ConfigsWithFields: map[string][]Variable{"ewc/broken.yaml": variables},
			},
			reason: "failed to process variables test",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			rec, err := covAPISaveAndStart(t, app, auth, s.protocol, s.version, s.input)
			requireHandlerErrorHandled(t, rec, err)
			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, s.reason, decodeHandlerErrorResponse(t, rec).Error.Reason)
		})
	}
}

func TestProcessVariablesTestSaveFailureStopsBeforeStart(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	app.OnRecordCreate("config_values").BindFunc(func(*core.RecordEvent) error {
		return errors.New("boom")
	})

	dir := t.TempDir()
	covAPIWriteTemplate(t, dir, "ewc/test-1.yaml", "foo: bar\n")

	origRegistry := workflowRegistry
	t.Cleanup(func() { workflowRegistry = origRegistry })
	started := false
	workflowRegistry = map[Author]WorkflowStarter{
		"ewc": func(WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
			started = true
			return workflowengine.WorkflowResult{}, nil
		},
	}

	_, err = processVariablesTest(
		app,
		"ewc/test-1.yaml",
		[]Variable{{FieldName: "foo", Value: "bar", CredimiID: "cred-1"}},
		"user@example.org",
		"ns",
		dir,
		map[string]any{},
		"ewc",
		"ewc",
		"v1",
		"User",
		"org-1",
	)
	covAPIRequireStarterError(t, err, "failed to save variable for test ewc/test-1.yaml")
	assert.False(t, started)
}

func TestHandleGetConformanceCheckDeeplinkMissingTemplate(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	t.Setenv("ROOT_DIR", t.TempDir())

	for _, suite := range []string{workflows.EWCSuite, workflows.WebuildSuite} {
		t.Run(suite, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				"/api/conformance-check/deeplink?id=openid4vp_wallet/"+suite+"/missing-check",
				nil,
			)
			rec := httptest.NewRecorder()
			err := HandleGetConformanceCheckDeeplink()(&core.RequestEvent{
				App:   app,
				Event: router.Event{Request: req, Response: rec},
			})
			requireHandlerErrorHandled(t, rec, err)
			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(
				t,
				"failed to read template file",
				decodeHandlerErrorResponse(t, rec).Error.Reason,
			)
		})
	}
}

func covAPIFidesEvent(
	t *testing.T,
	app core.App,
	auth *core.Record,
	body string,
) *core.RequestEvent {
	t.Helper()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/import-fides",
		strings.NewReader(body),
	)
	return &core.RequestEvent{
		App:   app,
		Auth:  auth,
		Event: router.Event{Request: req, Response: httptest.NewRecorder()},
	}
}

func TestHandleCredentialIssuerImportFidesFailures(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origRead := credentialIssuerReadSchemaFile
	origStart := fidesCredentialIssuersStartWorkflow
	origClient := fidesCredentialIssuersTemporalClient
	t.Cleanup(func() {
		credentialIssuerReadSchemaFile = origRead
		fidesCredentialIssuersStartWorkflow = origStart
		fidesCredentialIssuersTemporalClient = origClient
	})
	fidesCredentialIssuersStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errCovAPIStart
	}
	fidesCredentialIssuersTemporalClient = func(string) (client.Client, error) {
		return nil, errCovAPIStart
	}

	schemaOK := func(string) (string, *apierror.APIError) { return `{}`, nil }
	scenarios := []struct {
		name   string
		auth   *core.Record
		body   string
		read   func(string) (string, *apierror.APIError)
		code   int
		reason string
	}{
		{name: "unauthenticated", code: http.StatusUnauthorized, reason: "authentication required"},
		{
			name:   "malformed body",
			auth:   auth,
			body:   "{",
			code:   http.StatusBadRequest,
			reason: "invalid_request",
		},
		{
			name:   "negative interval",
			auth:   auth,
			body:   `{"interval_days":-1}`,
			code:   http.StatusBadRequest,
			reason: "invalid_request",
		},
		{
			name: "schema unreadable",
			auth: auth,
			read: func(string) (string, *apierror.APIError) {
				return "", apierror.New(
					http.StatusInternalServerError,
					"schema",
					"schema missing",
					"x",
				)
			},
			code:   http.StatusInternalServerError,
			reason: "schema missing",
		},
		{
			name:   "immediate start fails",
			auth:   auth,
			read:   schemaOK,
			code:   http.StatusInternalServerError,
			reason: "failed to start Fides credential issuers import",
		},
		{
			name:   "schedule fails",
			auth:   auth,
			body:   `{"interval_days":2}`,
			read:   schemaOK,
			code:   http.StatusInternalServerError,
			reason: "failed to schedule Fides credential issuers import",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			credentialIssuerReadSchemaFile = s.read
			err := HandleCredentialIssuerImportFides()(covAPIFidesEvent(t, app, s.auth, s.body))
			covAPIRequireAPIError(t, err, s.code, s.reason)
		})
	}
}

func TestScheduleFidesCredentialIssuersImportUpsertsExistingSchedule(t *testing.T) {
	input := workflowengine.WorkflowInput{Config: map[string]any{"orgID": "org-1"}}

	scenarios := []struct {
		name        string
		updateErr   error
		wantTrigger bool
		wantErr     string
	}{
		{name: "existing schedule is updated and triggered", wantTrigger: true},
		{
			name:      "update failure",
			updateErr: errors.New("update rejected"),
			wantErr:   "failed to upsert Fides import schedule: update rejected",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			handle := temporalmocks.NewScheduleHandle(t)
			var updated *client.ScheduleUpdate
			handle.On("Update", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					opts := args.Get(1).(client.ScheduleUpdateOptions)
					var err error
					updated, err = opts.DoUpdate(client.ScheduleUpdateInput{})
					require.NoError(t, err)
				}).
				Return(s.updateErr).Once()
			if s.wantTrigger {
				handle.On("Trigger", mock.Anything, fidesCredentialIssuersScheduleTriggerOptions).
					Return(nil).Once()
			}
			schedules := &fakeScheduleClient{
				handle:    handle,
				createErr: &serviceerror.AlreadyExists{Message: "exists"},
			}
			m := temporalmocks.NewClient(t)
			m.On("ScheduleClient").Return(schedules)

			orig := fidesCredentialIssuersTemporalClient
			t.Cleanup(func() { fidesCredentialIssuersTemporalClient = orig })
			fidesCredentialIssuersTemporalClient = func(string) (client.Client, error) { return m, nil }

			result, err := scheduleFidesCredentialIssuersImport(
				context.Background(),
				"org-ns",
				input,
				2,
			)
			require.NotNil(t, updated)
			require.Len(t, updated.Schedule.Spec.Intervals, 1)
			assert.Equal(t, 48*time.Hour, updated.Schedule.Spec.Intervals[0].Every)
			action, ok := updated.Schedule.Action.(*client.ScheduleWorkflowAction)
			require.True(t, ok)
			assert.Equal(t, []any{input}, action.Args)

			if s.wantErr != "" {
				require.EqualError(t, err, s.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "org-ns", result["workflowNamespace"])
		})
	}
}

func TestScheduleFidesCredentialIssuersImportCreateFailure(t *testing.T) {
	schedules := &fakeScheduleClient{createErr: errors.New("quota exceeded")}
	m := temporalmocks.NewClient(t)
	m.On("ScheduleClient").Return(schedules)

	orig := fidesCredentialIssuersTemporalClient
	t.Cleanup(func() { fidesCredentialIssuersTemporalClient = orig })
	fidesCredentialIssuersTemporalClient = func(string) (client.Client, error) { return m, nil }

	_, err := scheduleFidesCredentialIssuersImport(
		context.Background(),
		"org-ns",
		workflowengine.WorkflowInput{},
		1,
	)
	require.EqualError(t, err, "failed to upsert Fides import schedule: quota exceeded")
	assert.Len(t, schedules.createdOptions, 1)
}
