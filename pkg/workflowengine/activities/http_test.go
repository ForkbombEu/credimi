// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestHTTPActivity_Execute(t *testing.T) {
	activity := NewHTTPActivity()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(activity.Execute)

	tests := []struct {
		name            string
		handlerFunc     http.HandlerFunc
		payload         HTTPActivityPayload
		expectError     bool
		expectedErrCode errorcodes.Code
		expectStatus    int
		expectResponse  any
	}{
		{
			name: "Success - GET request without headers/body",
			handlerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"message": "ok"}`))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "", // Set dynamically
			},
			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"message": "ok"},
		},
		{
			name: "Success - POST request with body and headers",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.Header().Set("X-Test", "value")
				var payload map[string]any
				json.NewDecoder(r.Body).Decode(&payload)
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]any{"received": payload["key"]})
			},
			payload: HTTPActivityPayload{
				Method: http.MethodPost,
				URL:    "",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]any{
					"key": "value",
				},
			},
			expectStatus:   http.StatusCreated,
			expectResponse: map[string]any{"received": "value"},
		},
		{
			name: "Failure - timeout",
			handlerFunc: func(_ http.ResponseWriter, _ *http.Request) {
				time.Sleep(2 * time.Second)
			},
			payload: HTTPActivityPayload{
				Method:  http.MethodGet,
				URL:     "",
				Timeout: "1",
			},
			expectError:     true,
			expectedErrCode: errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed],
		},
		{
			name: "Success - non-JSON response is returned as string",
			handlerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.Write([]byte("plain response"))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
			},
			expectStatus:   http.StatusOK,
			expectResponse: "plain response",
		},
		{
			name: "Success - GET request with query parameters",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				require.Equal(t, "value", query.Get("key"))
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"message": "query received"}`))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
				QueryParams: map[string]string{
					"key": "value",
				},
			},
			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"message": "query received"},
		},
		{
			name: "Failure - unexpected status code",
			handlerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			payload: HTTPActivityPayload{
				ExpectedStatus: http.StatusOK,
				Method:         http.MethodGet,
				URL:            "",
			},
			expectError:     true,
			expectedErrCode: errorcodes.Codes[errorcodes.UnexpectedHTTPStatusCode],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handlerFunc != nil {
				server := httptest.NewServer(tt.handlerFunc)
				defer server.Close()
				tt.payload.URL = server.URL
			}

			a := HTTPActivity{}
			var result workflowengine.ActivityResult
			input := workflowengine.ActivityInput{
				Payload: tt.payload,
			}
			future, err := env.ExecuteActivity(a.Execute, input)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrCode.Code)
				require.Contains(t, err.Error(), tt.expectedErrCode.Description)
			} else {
				require.NoError(t, err)
				future.Get(&result)
				require.Equal(t, tt.expectStatus, int(result.Output.(map[string]any)["status"].(float64)))
				require.Equal(t, tt.expectResponse, result.Output.(map[string]any)["body"])
			}
		})
	}
}

func TestHTTPActivity_WithOutputRules(t *testing.T) {
	activity := NewHTTPActivity()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(activity.Execute)

	tests := []struct {
		name           string
		handlerFunc    http.HandlerFunc
		payload        HTTPActivityPayload
		expectError    bool
		expectedStatus int
		expectedOutput map[string]any
	}{
		{
			name: "Success - Extract value with XPath",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				htmlResponse := `
				<html>
					<body>
						<div id="user-id">12345</div>
						<span class="token">abc123</span>
					</body>
				</html>`
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(htmlResponse))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
				Outputs: map[string]OutputRule{
					"user_id": {
						XPath: "//div[@id='user-id']/text()",
					},
					"token": {
						XPath: "//span[@class='token']/text()",
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectedOutput: map[string]any{
				"user_id": "12345",
				"token":   "abc123",
			},
		},
		{
			name: "Success - Extract value with CSS Selector",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				htmlResponse := `
				<html>
					<body>
						<h1 class="title">Welcome Home</h1>
						<p id="description">This is a test page</p>
					</body>
				</html>`
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(htmlResponse))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
				Outputs: map[string]OutputRule{
					"title": {
						Selector: "h1.title",
					},
					"description": {
						Selector: "p#description",
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectedOutput: map[string]any{
				"title":       "Welcome Home",
				"description": "This is a test page",
			},
		},
		{
			name: "Success - Extract value with Regex",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				response := `
				authenticationRequest: 'openid://auth?code=xyz789'
				some other text
				token: abc-123-def
				`
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(response))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
				Outputs: map[string]OutputRule{
					"auth_code": {
						Regex: `authenticationRequest:\s*'([^']+)'`,
					},
					"token": {
						Regex: `token:\s*([a-z0-9-]+)`,
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectedOutput: map[string]any{
				"auth_code": "openid://auth?code=xyz789",
				"token":     "abc-123-def",
			},
		},
		{
			name: "Success - Extract value from Cookie",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				http.SetCookie(w, &http.Cookie{
					Name:  "session_id",
					Value: "sess_67890",
				})
				http.SetCookie(w, &http.Cookie{
					Name:  "csrf_token",
					Value: "csrf_abc123",
				})
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("<html><body>OK</body></html>"))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
				Outputs: map[string]OutputRule{
					"session": {
						Cookie: "session_id",
					},
					"csrf": {
						Cookie: "csrf_token",
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectedOutput: map[string]any{
				"session": "sess_67890",
				"csrf":    "csrf_abc123",
			},
		},
		{
			name: "Success - Mixed output rules (XPath + Regex + Cookie)",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				htmlResponse := `
				<html>
					<body>
						<div class="user-info">John Doe</div>
						<script>var deeplink = 'openid://app?code=test123';</script>
					</body>
				</html>`
				http.SetCookie(w, &http.Cookie{
					Name:  "session",
					Value: "mixed_test_session",
				})
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(htmlResponse))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
				Outputs: map[string]OutputRule{
					"username": {
						XPath: "//div[@class='user-info']/text()",
					},
					"deeplink": {
						Regex: `deeplink\s*=\s*'([^']+)'`,
					},
					"session": {
						Cookie: "session",
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectedOutput: map[string]any{
				"username": "John Doe",
				"deeplink": "openid://app?code=test123",
				"session":  "mixed_test_session",
			},
		},
		{
			name: "Success - XPath not found returns empty string",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				htmlResponse := `<html><body><div>Hello</div></body></html>`
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(htmlResponse))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
				Outputs: map[string]OutputRule{
					"missing": {
						XPath: "//div[@id='not-exists']/text()",
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectedOutput: map[string]any{
				"missing": "",
			},
		},
		{
			name: "Success - Regex not found returns empty string",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("some random text without pattern"))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
				Outputs: map[string]OutputRule{
					"missing": {
						Regex: `pattern_not_found: '([^']+)'`,
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectedOutput: map[string]any{
				"missing": "",
			},
		},
		{
			name: "Error - XPath on non-HTML content",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message": "json response"}`))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
				Outputs: map[string]OutputRule{
					"value": {
						XPath: "//some/path",
					},
				},
			},
			expectError:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Error - Invalid regex pattern",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("some response"))
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "",
				Outputs: map[string]OutputRule{
					"value": {
						Regex: `[invalid regex`,
					},
				},
			},
			expectError:    true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handlerFunc)
			defer server.Close()
			tt.payload.URL = server.URL

			var result workflowengine.ActivityResult
			input := workflowengine.ActivityInput{
				Payload: tt.payload,
			}
			future, err := env.ExecuteActivity(activity.Execute, input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				err = future.Get(&result)
				require.NoError(t, err)

				outputMap := result.Output.(map[string]any)
				require.Equal(t, tt.expectedStatus, int(outputMap["status"].(float64)))

				// Verifica gli output estratti
				extractedOutputs, ok := outputMap["outputs"].(map[string]any)
				require.True(t, ok, "outputs should be present in result")

				for key, expectedValue := range tt.expectedOutput {
					actualValue, exists := extractedOutputs[key]
					require.True(t, exists, "key %s should exist in outputs", key)
					require.Equal(t, expectedValue, actualValue, "value mismatch for key %s", key)
				}
			}
		})
	}
}
