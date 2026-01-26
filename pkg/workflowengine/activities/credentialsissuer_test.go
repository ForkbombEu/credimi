// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestCheckCredentialsIssuerActivity_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()

	act := NewCheckCredentialsIssuerActivity()
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{
		Name: act.Name(),
	})

	tests := []struct {
		name            string
		payload         CheckCredentialsIssuerActivityPayload
		responseStatus  int
		responseBody    string
		transportErr    error
		bodyReadFailure bool
		expectErr       bool
		expectedErrCode errorcodes.Code
		expectedOutput  map[string]any
	}{
		{
			name: "Success - valid issuer response",
			payload: CheckCredentialsIssuerActivityPayload{
				BaseURL: "https://issuer.test",
			},
			responseStatus: http.StatusOK,
			responseBody:   `{"issuer":"example.com"}`,
			expectErr:      false,
			expectedOutput: map[string]any{
				"rawJSON": `{"issuer":"example.com"}`,
			},
		},
		{
			name:            "Failure - missing base_url",
			payload:         CheckCredentialsIssuerActivityPayload{},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure - non-200 status code",
			payload: CheckCredentialsIssuerActivityPayload{
				BaseURL: "https://issuer.test",
			},
			responseStatus:  http.StatusForbidden,
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.IsNotCredentialIssuer],
		},
		{
			name: "Failure - error reaching issuer",
			payload: CheckCredentialsIssuerActivityPayload{
				BaseURL: "https://issuer.test",
			},
			transportErr:    errors.New("network error"),
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed],
		},
		{
			name: "Failure - error reading body",
			payload: CheckCredentialsIssuerActivityPayload{
				BaseURL: "https://issuer.test",
			},
			responseStatus:  http.StatusOK,
			bodyReadFailure: true,
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.ReadFromReaderFailed],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.payload.BaseURL != "" {
				withMockTransport(t, func(req *http.Request) (*http.Response, error) {
					switch req.URL.Path {
					case "/.well-known/openid-federation":
						return &http.Response{
							StatusCode: http.StatusNotFound,
							Body:       io.NopCloser(strings.NewReader("not found")),
							Header:     http.Header{"Content-Type": []string{"text/plain"}},
						}, nil
					case "/.well-known/openid-credential-issuer":
						require.Equal(t, "/.well-known/openid-credential-issuer", req.URL.Path)
						if tt.transportErr != nil {
							return nil, tt.transportErr
						}
						if tt.bodyReadFailure {
							return &http.Response{
								StatusCode: tt.responseStatus,
								Body:       errorReadCloser{err: errors.New("read error")},
								Header:     http.Header{"Content-Type": []string{"application/json"}},
							}, nil
						}
						return &http.Response{
							StatusCode: tt.responseStatus,
							Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
							Header:     http.Header{"Content-Type": []string{"application/json"}},
						}, nil
					default:
						return &http.Response{
							StatusCode: http.StatusNotFound,
							Body:       io.NopCloser(strings.NewReader("not found")),
							Header:     http.Header{"Content-Type": []string{"text/plain"}},
						}, nil
					}
				})
			}

			input := workflowengine.ActivityInput{
				Payload: tt.payload,
			}

			future, err := env.ExecuteActivity(act.Execute, input)

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrCode.Code)
				require.Contains(t, err.Error(), tt.expectedErrCode.Description)
			} else {
				require.NoError(t, err)
				var result workflowengine.ActivityResult
				require.NoError(t, future.Get(&result))
				for k, v := range tt.expectedOutput {
					require.Equal(t, v, result.Output.(map[string]any)[k])
				}
				require.Contains(t, result.Output.(map[string]any)["base_url"], tt.payload.BaseURL)
			}
		})
	}
}

type errorReadCloser struct {
	err error
}

func (reader errorReadCloser) Read(_ []byte) (int, error) {
	return 0, reader.err
}

func (reader errorReadCloser) Close() error {
	return nil
}
