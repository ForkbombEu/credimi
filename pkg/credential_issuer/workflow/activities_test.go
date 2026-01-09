// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/testsuite"
	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/lib"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestFetchIssuersActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(FetchIssuersActivity)

	originalClient := fidesHTTPClient
	fidesHTTPClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			body := `{"content":[{"issuanceUrl":"https://example.com/issuer/.well-known/openid-credential-issuer"}],` +
				`"page":{"size":1,"number":0,"totalElements":1,"totalPages":0}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() {
		fidesHTTPClient = originalClient
	})

	val, err := env.ExecuteActivity(FetchIssuersActivity)
	var result FetchIssuersActivityResponse
	assert.NoError(t, val.Get(&result))
	assert.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/issuer"}, result.Issuers)
}

func TestExtractHrefsFromApiResponse(t *testing.T) {
	root := FidesResponse{
		Content: []struct {
			IssuanceURL               string `json:"issuanceUrl"`
			CredentialConfigurationID string `json:"credentialConfigurationId"`
			IssuePortalURL            string `json:"issuePortalUrl,omitempty"`
		}{
			{
				IssuanceURL: "https://example.com/123/.well-known/openid-credential-issuer",
			},
			{
				IssuanceURL: "https://example.com/456",
			},
		},
		Page: struct {
			Size          int `json:"size"`
			Number        int `json:"number"`
			TotalElements int `json:"totalElements"`
			TotalPages    int `json:"totalPages"`
		}{
			Number: 0,
		},
	}

	hrefs := extractHrefsFromAPIResponse(root)
	assert.Equal(t, []string{"https://example.com/123", "https://example.com/456"}, hrefs)
}

func TestRemoveWellKnownSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with suffix",
			input:    "https://wallet.acc.credenco.com/public/c497db8f-4906-4a8e-96e1-e52927166e07/credencoInjiIssuer/.well-known/openid-credential-issuer",
			expected: "https://wallet.acc.credenco.com/public/c497db8f-4906-4a8e-96e1-e52927166e07/credencoInjiIssuer",
		},
		{
			name:     "URL without suffix",
			input:    "https://wallet.acc.credenco.com/public/c497db8f-4906-4a8e-96e1-e52927166e07/credencoInjiIssuer",
			expected: "https://wallet.acc.credenco.com/public/c497db8f-4906-4a8e-96e1-e52927166e07/credencoInjiIssuer",
		},
		{
			name:     "URL with a different well-known segment",
			input:    "https://example.com/path/.well-known/some-other-value",
			expected: "https://example.com/path/.well-known/some-other-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeWellKnownSuffix(tt.input)
			if result != tt.expected {
				t.Errorf(
					"RemoveWellKnownSuffix(%q) = %q; expected %q",
					tt.input,
					result,
					tt.expected,
				)
			}
		})
	}
}
