// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/safehttp"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

const testInternalAdminKey = "internal-admin-key"

// runnerServer counts the connections and requests a test runner receives.
type runnerServer struct {
	*httptest.Server
	conns    atomic.Int32
	requests atomic.Int32
	path     atomic.Value
	key      atomic.Value
}

func newRunnerServer(t *testing.T, useTLS bool) *runnerServer {
	t.Helper()
	server := &runnerServer{}
	server.Server = httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			server.requests.Add(1)
			server.path.Store(r.URL.Path)
			server.key.Store(r.Header.Get("Credimi-Api-Key"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}),
	)
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			server.conns.Add(1)
		}
	}
	if useTLS {
		server.StartTLS()
	} else {
		server.Start()
	}
	t.Cleanup(server.Close)
	return server
}

func createHTTPTestRunner(
	t *testing.T,
	app *tests.TestApp,
	name string,
	runnerURL string,
	adminManaged bool,
) (*core.Record, string) {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(coll)
	runner.Set("owner", testOrgAID)
	runner.Set("name", name)
	runner.Set("ip", runnerURL)
	runner.Set("type", "android_emulator")
	runner.Set("admin_managed", adminManaged)
	runner.Set("credential_generation", 4)
	require.NoError(t, app.Save(runner))
	runnerID, err := mobilerunner.RunnerIdentifier(app, runner)
	require.NoError(t, err)
	return runner, runnerID
}

func setRunnerHTTPTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", testInternalAdminKey)
	t.Setenv(mobilerunner.CredentialSecretEnvVar, "runner-credential-secret")
}

func requireNonRetryable(t *testing.T, err error) {
	t.Helper()
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "want a Temporal application error, got %v", err)
	require.True(t, appErr.NonRetryable())
}

// The runner is reached at its stored address with its own credential, never
// with the internal admin key, and a caller cannot redirect the call.
func TestMobileRunnerHTTPActivityCallsStoredRunnerWithItsCredential(t *testing.T) {
	setRunnerHTTPTestEnv(t)
	app := newCredimiTestApp(t)
	runnerSrv := newRunnerServer(t, false)
	elsewhere := newRunnerServer(t, false)
	runner, runnerID := createHTTPTestRunner(t, app, "Admin Runner", runnerSrv.URL, true)
	wantKey, err := mobilerunner.Credential(runner)
	require.NoError(t, err)

	res, err := NewMobileRunnerHTTPActivity(app).Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: map[string]any{
			"method":    http.MethodPost,
			"runner_id": runnerID,
			"path":      "/credimi/installer-action",
			// Not a payload field: the destination always comes from the record.
			"url":             elsewhere.URL,
			"expected_status": http.StatusOK,
		}},
	)
	require.NoError(t, err)
	output, ok := res.Output.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.StatusOK, output["status"])

	require.EqualValues(t, 1, runnerSrv.requests.Load())
	require.Equal(t, "/credimi/installer-action", runnerSrv.path.Load())
	require.Equal(t, wantKey, runnerSrv.key.Load())
	require.NotEqual(t, testInternalAdminKey, runnerSrv.key.Load())
	require.Zero(t, elsewhere.conns.Load())
}

// A tenant chooses its runner's address, so a tenant runner on Credimi's own
// network must never be contacted.
func TestMobileRunnerHTTPActivityRefusesTenantDestinations(t *testing.T) {
	for _, tc := range []struct {
		name   string
		useTLS bool
	}{
		{name: "tenant runner over http"},
		{name: "tenant runner at a loopback address", useTLS: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setRunnerHTTPTestEnv(t)
			app := newCredimiTestApp(t)
			runnerSrv := newRunnerServer(t, tc.useTLS)
			_, runnerID := createHTTPTestRunner(t, app, "Tenant Runner", runnerSrv.URL, false)

			_, err := NewMobileRunnerHTTPActivity(app).Execute(
				context.Background(),
				workflowengine.ActivityInput{Payload: MobileRunnerHTTPActivityPayload{
					Method:   http.MethodPost,
					RunnerID: runnerID,
					Path:     "/credimi/pipeline-result",
				}},
			)
			require.ErrorIs(t, err, safehttp.ErrBlockedDestination)
			requireNonRetryable(t, err)
			require.Zero(t, runnerSrv.conns.Load(), "the runner must never be contacted")
			require.Zero(t, runnerSrv.requests.Load())
		})
	}
}

func TestMobileRunnerHTTPActivityFailures(t *testing.T) {
	for _, tc := range []struct {
		name        string
		secret      string
		runnerID    func(string) string
		wantCode    string
		wantMessage string
		wantNoRetry bool
	}{
		{
			name:        "missing credential secret",
			runnerID:    func(id string) string { return id },
			wantCode:    errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
			wantMessage: mobilerunner.CredentialSecretEnvVar,
		},
		{
			name:        "unknown runner",
			secret:      "runner-credential-secret",
			runnerID:    func(id string) string { return id + "-missing" },
			wantCode:    errorcodes.Codes[errorcodes.RecordNotFound].Code,
			wantMessage: "mobile runner not found",
			wantNoRetry: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", testInternalAdminKey)
			t.Setenv(mobilerunner.CredentialSecretEnvVar, tc.secret)
			app := newCredimiTestApp(t)
			runnerSrv := newRunnerServer(t, false)
			_, runnerID := createHTTPTestRunner(t, app, "Admin Runner", runnerSrv.URL, true)

			_, err := NewMobileRunnerHTTPActivity(app).Execute(
				context.Background(),
				workflowengine.ActivityInput{Payload: MobileRunnerHTTPActivityPayload{
					Method:   http.MethodGet,
					RunnerID: tc.runnerID(runnerID),
					Path:     "/credimi/live-view",
				}},
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantCode)
			require.Contains(t, err.Error(), tc.wantMessage)
			if tc.wantNoRetry {
				requireNonRetryable(t, err)
			}
			require.Zero(t, runnerSrv.requests.Load())
		})
	}
}

func TestMobileRunnerHTTPActivityRequiresRunnerAndPath(t *testing.T) {
	for _, payload := range []MobileRunnerHTTPActivityPayload{
		{Method: http.MethodGet, Path: "/health"},
		{Method: http.MethodGet, RunnerID: "org/runner"},
	} {
		_, err := NewMobileRunnerHTTPActivity(nil).Execute(
			context.Background(),
			workflowengine.ActivityInput{Payload: payload},
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)
	}
}

func TestMobileRunnerHTTPActivityOmittedExpectedStatusDoesNotExpectZero(t *testing.T) {
	setRunnerHTTPTestEnv(t)
	app := newCredimiTestApp(t)
	runnerSrv := newRunnerServer(t, false)
	_, runnerID := createHTTPTestRunner(t, app, "Admin Runner", runnerSrv.URL, true)

	res, err := NewMobileRunnerHTTPActivity(
		app,
	).Execute(context.Background(), workflowengine.ActivityInput{
		Payload: MobileRunnerHTTPActivityPayload{
			Method:   http.MethodGet,
			RunnerID: runnerID,
			Path:     "/health",
		},
	})
	require.NoError(t, err)
	output, ok := res.Output.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.StatusOK, output["status"])
}
