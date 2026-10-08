// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func TestOpenIDNetLogsActivityUsesTokenFromEnv(t *testing.T) {
	const token = "env-secret-token"
	t.Setenv("OPENIDNET_TOKEN", token)

	var gotAuth, gotPath, gotPublic string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotPublic = r.URL.Query().Get("public")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"result":"RUNNING"}]`))
	}))
	defer server.Close()

	origBaseURL := openIDNetLogsBaseURL
	openIDNetLogsBaseURL = server.URL + "/api/log"
	t.Cleanup(func() { openIDNetLogsBaseURL = origBaseURL })

	act := NewOpenIDNetLogsActivity()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(act.Execute)

	input := workflowengine.ActivityInput{Payload: OpenIDNetLogsPayload{Rid: "run-1"}}
	encodedInput, err := json.Marshal(input)
	require.NoError(t, err)
	require.NotContains(t, string(encodedInput), token)

	future, err := env.ExecuteActivity(act.Execute, input)
	require.NoError(t, err)

	var result workflowengine.ActivityResult
	require.NoError(t, future.Get(&result))
	require.Equal(t, "Bearer "+token, gotAuth)
	require.Equal(t, "/api/log/run-1", gotPath)
	require.Equal(t, "false", gotPublic)
	require.Equal(
		t,
		[]map[string]any{{"result": "RUNNING"}},
		workflowengine.AsSliceOfMaps(workflowengine.AsMap(result.Output)["body"]),
	)
}

func TestOpenIDNetLogsActivityMissingToken(t *testing.T) {
	t.Setenv("OPENIDNET_TOKEN", "")

	act := NewOpenIDNetLogsActivity()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(act.Execute)

	_, err := env.ExecuteActivity(
		act.Execute,
		workflowengine.ActivityInput{Payload: OpenIDNetLogsPayload{Rid: "run-1"}},
	)
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.True(t, appErr.NonRetryable())
	require.Equal(t, errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code, appErr.Type())
}
