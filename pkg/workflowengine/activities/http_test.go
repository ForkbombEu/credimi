// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func setHTTPClientFactory(
	t testing.TB,
	handler func(req *http.Request) (*http.Response, error),
) {
	t.Helper()
	original := httpClientFactory
	httpClientFactory = func(_ time.Duration) httpDoer {
		return &http.Client{Transport: roundTripperFunc(handler)}
	}
	t.Cleanup(func() {
		httpClientFactory = original
	})
}

func TestHTTPActivity_Execute(t *testing.T) {
	activity := NewHTTPActivity()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(activity.Execute)

	tests := []struct {
		name            string
		handlerFunc     func(req *http.Request) (*http.Response, error)
		payload         HTTPActivityPayload
		expectError     bool
		expectedErrCode errorcodes.Code
		expectStatus    int
		expectResponse  any
	}{
		{
			name: "Success - GET request without headers/body",
			handlerFunc: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"message": "ok"}`)),
					Header:     make(http.Header),
				}, nil
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "https://example.com",
			},
			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"message": "ok"},
		},
		{
			name: "Success - POST request with body and headers",
			handlerFunc: func(r *http.Request) (*http.Response, error) {
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				var payload map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				respBody, _ := json.Marshal(map[string]any{"received": payload["key"]})
				header := make(http.Header)
				header.Set("X-Test", "value")
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(string(respBody))),
					Header:     header,
				}, nil
			},
			payload: HTTPActivityPayload{
				Method: http.MethodPost,
				URL:    "https://example.com",
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
			handlerFunc: func(_ *http.Request) (*http.Response, error) {
				return nil, context.DeadlineExceeded
			},
			payload: HTTPActivityPayload{
				Method:  http.MethodGet,
				URL:     "https://example.com",
				Timeout: "1",
			},
			expectError:     true,
			expectedErrCode: errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed],
		},
		{
			name: "Success - non-JSON response is returned as string",
			handlerFunc: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("plain response")),
					Header:     make(http.Header),
				}, nil
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "https://example.com",
			},
			expectStatus:   http.StatusOK,
			expectResponse: "plain response",
		},
		{
			name: "Success - GET request with query parameters",
			handlerFunc: func(r *http.Request) (*http.Response, error) {
				query := r.URL.Query()
				require.Equal(t, "value", query.Get("key"))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"message": "query received"}`)),
					Header:     make(http.Header),
				}, nil
			},
			payload: HTTPActivityPayload{
				Method: http.MethodGet,
				URL:    "https://example.com",
				QueryParams: map[string]string{
					"key": "value",
				},
			},
			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"message": "query received"},
		},
		{
			name: "Failure - unexpected status code",
			handlerFunc: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
			payload: HTTPActivityPayload{
				ExpectedStatus: http.StatusOK,
				Method:         http.MethodGet,
				URL:            "https://example.com",
			},
			expectError:     true,
			expectedErrCode: errorcodes.Codes[errorcodes.UnexpectedHTTPStatusCode],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setHTTPClientFactory(t, func(req *http.Request) (*http.Response, error) {
				if tt.handlerFunc == nil {
					return nil, context.Canceled
				}
				return tt.handlerFunc(req)
			})

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
