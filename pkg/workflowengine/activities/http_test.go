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
		name           string
		handlerFunc    http.HandlerFunc
		input          workflowengine.ActivityInput
		expectError    bool
		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success - GET request without headers/body",
			handlerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"message": "ok"}`))
			},
			input: workflowengine.ActivityInput{
				Config: map[string]string{
					"method": "GET",
					"url":    "", // Set dynamically
				},
				Payload: map[string]any{},
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
			input: workflowengine.ActivityInput{
				Config: map[string]string{
					"method": "POST",
					"url":    "",
				},
				Payload: map[string]any{
					"headers": map[string]any{
						"Content-Type": "application/json",
					},
					"body": map[string]any{
						"key": "value",
					},
				},
			},
			expectStatus:   http.StatusCreated,
			expectResponse: map[string]any{"received": "value"},
		},
		{
			name: "Failure - invalid method",
			input: workflowengine.ActivityInput{
				Config: map[string]string{
					"method": "",
					"url":    "http://example.com",
				},
			},
			expectError: true,
		},
		{
			name: "Failure - timeout",
			handlerFunc: func(_ http.ResponseWriter, _ *http.Request) {
				time.Sleep(2 * time.Second)
			},
			input: workflowengine.ActivityInput{
				Config: map[string]string{
					"method":  "GET",
					"url":     "",
					"timeout": "1",
				},
			},
			expectError: true,
		},
		{
			name: "Success - non-JSON response is returned as string",
			handlerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.Write([]byte("plain response"))
			},
			input: workflowengine.ActivityInput{
				Config: map[string]string{
					"method": "GET",
					"url":    "",
				},
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
			input: workflowengine.ActivityInput{
				Config: map[string]string{
					"method": "GET",
					"url":    "", // Set dynamically
				},
				Payload: map[string]any{
					"query_params": map[string]any{
						"key": "value",
					},
				},
			},
			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"message": "query received"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handlerFunc != nil {
				server := httptest.NewServer(tt.handlerFunc)
				defer server.Close()
				tt.input.Config["url"] = server.URL
			}

			a := HTTPActivity{}
			var result workflowengine.ActivityResult
			future, err := env.ExecuteActivity(a.Execute, tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				future.Get(&result)
				require.Equal(t, tt.expectStatus, int(result.Output.(map[string]any)["status"].(float64)))
				require.Equal(t, tt.expectResponse, result.Output.(map[string]any)["body"])
			}
		})
	}
}
