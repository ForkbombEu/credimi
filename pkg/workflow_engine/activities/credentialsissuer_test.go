// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	workflowengine "github.com/forkbombeu/didimo/pkg/workflow_engine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestCheckCredentialsIssuerActivity_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()

	activity := &CheckCredentialsIssuerActivity{}
	env.RegisterActivity(activity.Execute)

	tests := []struct {
		name           string
		config         map[string]string
		serverHandler  http.HandlerFunc
		expectErr      bool
		expectedErrMsg string
		expectedOutput map[string]any
	}{
		{
			name: "Success - valid issuer response",
			config: map[string]string{
				"base_url": "",
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/.well-known/openid-credential-issuer", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"issuer":"example.com"}`)
			},
			expectErr: false,
			expectedOutput: map[string]any{
				"rawJSON": `{"issuer":"example.com"}`,
			},
		},
		{
			name:           "Failure - missing base_url",
			config:         map[string]string{},
			expectErr:      true,
			expectedErrMsg: "Missing baseURL in config",
		},
		{
			name:           "Failure - empty base_url",
			config:         map[string]string{"base_url": "  "},
			expectErr:      true,
			expectedErrMsg: "Missing baseURL in config",
		},
		{
			name: "Failure - non-200 status code",
			config: map[string]string{
				"base_url": "",
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			expectErr:      true,
			expectedErrMsg: "Not a credential issuer",
		},
		{
			name: "Failure - error reaching issuer",
			config: map[string]string{
				"base_url": "",
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "10")
				conn, _, _ := w.(http.Hijacker).Hijack()
				conn.Close()
			},
			expectErr:      true,
			expectedErrMsg: "Could not reach issue",
		},
		{
			name: "Failure - error reading body",
			config: map[string]string{
				"base_url": "",
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"partial":`))
				conn, _, _ := w.(http.Hijacker).Hijack()
				conn.Close() // simulate read failure
			},
			expectErr:      true,
			expectedErrMsg: "Error reading response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var baseURL string
			if tt.serverHandler != nil {
				server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
				defer server.Close()
				baseURL = server.URL
				tt.config["base_url"] = strings.TrimPrefix(baseURL, "https://")
			}

			input := workflowengine.ActivityInput{
				Config: tt.config,
			}

			future, err := env.ExecuteActivity(activity.Execute, input)

			if tt.expectErr {
				require.Error(t, err)
				if tt.expectedErrMsg != "" {
					require.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				require.NoError(t, err)
				var result workflowengine.ActivityResult
				require.NoError(t, future.Get(&result))
				for k, v := range tt.expectedOutput {
					require.Equal(t, v, result.Output.(map[string]any)[k])
				}
				require.Contains(t, result.Output.(map[string]any)["base_url"], baseURL)
			}
		})
	}
}
