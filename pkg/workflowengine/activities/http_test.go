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
