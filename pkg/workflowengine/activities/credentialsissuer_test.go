// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"errors"
	"fmt"
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

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func setIssuerHTTPClient(
	t testing.TB,
	handler func(req *http.Request) (*http.Response, error),
) {
	t.Helper()
	original := issuerHTTPClient
	issuerHTTPClient = &http.Client{
		Transport: roundTripperFunc(handler),
	}
	t.Cleanup(func() {
		issuerHTTPClient = original
	})
}

type errorReadCloser struct {
	err error
}

func (reader *errorReadCloser) Read(_ []byte) (int, error) {
	return 0, reader.err
}

func (reader *errorReadCloser) Close() error {
	return nil
}

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
		responseHandler func(req *http.Request) (*http.Response, error)
		expectErr       bool
		expectedErrCode errorcodes.Code
		expectedOutput  map[string]any
	}{
		{
			name: "Success - valid issuer response",
			payload: CheckCredentialsIssuerActivityPayload{
				BaseURL: "https://issuer.example.com/.well-known/openid-credential-issuer",
			},
			responseHandler: func(r *http.Request) (*http.Response, error) {
				require.Equal(t, "/.well-known/openid-credential-issuer", r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"issuer":"example.com"}`)),
					Header:     make(http.Header),
				}, nil
			},
			expectErr: false,
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
				BaseURL: "https://issuer.example.com/.well-known/openid-credential-issuer",
			},
			responseHandler: func(r *http.Request) (*http.Response, error) {
				require.Equal(t, "/.well-known/openid-credential-issuer", r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.IsNotCredentialIssuer],
		},
		{
			name: "Failure - error reaching issuer",
			payload: CheckCredentialsIssuerActivityPayload{
				BaseURL: "https://issuer.example.com/.well-known/openid-credential-issuer",
			},
			responseHandler: func(r *http.Request) (*http.Response, error) {
				require.Equal(t, "/.well-known/openid-credential-issuer", r.URL.Path)
				return nil, errors.New("connection failed")
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed],
		},
		{
			name: "Failure - error reading body",
			payload: CheckCredentialsIssuerActivityPayload{
				BaseURL: "https://issuer.example.com/.well-known/openid-credential-issuer",
			},
			responseHandler: func(r *http.Request) (*http.Response, error) {
				require.Equal(t, "/.well-known/openid-credential-issuer", r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       &errorReadCloser{err: fmt.Errorf("read failed")},
					Header:     make(http.Header),
				}, nil
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.ReadFromReaderFailed],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setIssuerHTTPClient(t, func(req *http.Request) (*http.Response, error) {
				if tt.responseHandler == nil {
					return nil, errors.New("unexpected request")
				}
				return tt.responseHandler(req)
			})

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
