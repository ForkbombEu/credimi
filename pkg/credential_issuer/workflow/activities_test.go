// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/lib"
)

func TestFetchIssuersActivity(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, FidesIssuersURL, req.URL.String())
			body := `{"content":[{"issuanceUrl":"https://example.com/issuer/.well-known/openid-credential-issuer"}],"page":{"number":0,"totalPages":0}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() {
		http.DefaultClient = origClient
	})

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(FetchIssuersActivity)

	val, err := env.ExecuteActivity(FetchIssuersActivity)
	var result FetchIssuersActivityResponse
	assert.NoError(t, val.Get(&result))
	assert.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/issuer"}, result.Issuers)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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

func TestFetchIssuersActivityNonOKResponse(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("boom")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() {
		http.DefaultClient = origClient
	})

	_, err := fetchIssuersRecursive(context.Background(), 0)
	require.Error(t, err)
}

func TestFetchIssuersActivityInvalidJSON(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{not-json")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() {
		http.DefaultClient = origClient
	})

	_, err := fetchIssuersRecursive(context.Background(), 0)
	require.Error(t, err)
}

func TestFetchIssuersActivityPagination(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			var (
				pageNumber int
				totalPages int
				count      int
			)

			if req.URL.String() == FidesIssuersURL {
				pageNumber = 0
				totalPages = 1
				count = 200
			} else {
				pageNumber = 1
				totalPages = 1
				count = 1
			}

			bodyBytes, err := json.Marshal(buildFidesResponse(pageNumber, totalPages, count))
			require.NoError(t, err)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(string(bodyBytes))),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() {
		http.DefaultClient = origClient
	})

	hrefs, err := fetchIssuersRecursive(context.Background(), 0)
	require.NoError(t, err)
	require.Len(t, hrefs, 201)
}

func buildFidesResponse(pageNumber, totalPages, count int) FidesResponse {
	root := FidesResponse{}
	root.Page.Number = pageNumber
	root.Page.TotalPages = totalPages
	root.Content = make([]struct {
		IssuanceURL               string `json:"issuanceUrl"`
		CredentialConfigurationID string `json:"credentialConfigurationId"`
		IssuePortalURL            string `json:"issuePortalUrl,omitempty"`
	}, count)
	for i := range root.Content {
		root.Content[i].IssuanceURL = "https://example.com/issuer/.well-known/openid-credential-issuer"
	}
	return root
}

func TestCreateCredentialIssuersActivityInsertsAndSkipsExisting(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	_, err = db.Exec("CREATE TABLE credential_issuers (url TEXT)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO credential_issuers(url) VALUES (?)", "https://issuer-1")
	require.NoError(t, err)

	input := CreateCredentialIssuersInput{
		Issuers: []string{"https://issuer-1", "https://issuer-2"},
		DBPath:  dbPath,
	}

	require.NoError(t, CreateCredentialIssuersActivity(context.Background(), input))

	rows, err := db.Query("SELECT url FROM credential_issuers ORDER BY url")
	require.NoError(t, rows.Err())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, rows.Close())
	})

	var urls []string
	for rows.Next() {
		var url string
		require.NoError(t, rows.Scan(&url))
		urls = append(urls, url)
	}
	require.Equal(t, []string{"https://issuer-1", "https://issuer-2"}, urls)
}

func TestCreateCredentialIssuersActivityMissingTable(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	input := CreateCredentialIssuersInput{
		Issuers: []string{"https://issuer-1"},
		DBPath:  dbPath,
	}

	err = CreateCredentialIssuersActivity(context.Background(), input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to check if issuer exists")
}
