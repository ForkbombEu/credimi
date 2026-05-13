// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build unit

package githubapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitHubAPIURL(t *testing.T) {
	client := &Client{apiURL: "https://api.github.com"}

	require.Equal(
		t,
		"https://api.github.com/repos/ForkbombEu/eudi-app-android-wallet-ui/installation",
		client.githubAPIURL("repos", "ForkbombEu", "eudi-app-android-wallet-ui", "installation"),
	)
	require.Equal(
		t,
		"https://api.github.com/repos/ForkbombEu/eudi-app-android-wallet-ui/issues/1/comments?per_page=100",
		withQueryParam(
			client.githubAPIURL("repos", "ForkbombEu", "eudi-app-android-wallet-ui", "issues", "1", "comments"),
			"per_page",
			"100",
		),
	)
}

func TestCreateOrUpdatePRCommentPatchesExistingComment(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	var patchedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.Header.Get("Authorization"), "Bearer ")
		switch r.URL.Path {
		case "/repos/acme/wallet/installation":
			require.Equal(t, http.MethodGet, r.Method)
			writeJSON(t, w, map[string]any{"id": 123})
		case "/app/installations/123/access_tokens":
			require.Equal(t, http.MethodPost, r.Method)
			writeJSON(t, w, map[string]any{"token": "installation-token"})
		case "/repos/acme/wallet/issues/7/comments":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "100", r.URL.Query().Get("per_page"))
			writeJSON(t, w, []map[string]any{{
				"id":   44,
				"body": "previous\n\n" + DefaultMarker,
			}})
		case "/repos/acme/wallet/issues/comments/44":
			require.Equal(t, http.MethodPatch, r.Method)
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			patchedBody = body["body"]
			writeJSON(t, w, map[string]any{"id": 44})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	client := &Client{
		httpClient: server.Client(),
		apiURL:     server.URL,
		clientID:   "client-1",
		privateKey: privateKey,
	}

	result, err := client.CreateOrUpdatePRComment(context.Background(), PRComment{
		Repository:        "https://github.com/acme/wallet",
		PullRequestNumber: 7,
		Marker:            DefaultMarker,
		Body:              "updated body",
	})

	require.NoError(t, err)
	require.Equal(t, int64(44), result.CommentID)
	require.Contains(t, patchedBody, "updated body")
	require.Contains(t, patchedBody, DefaultMarker)
}

func TestCreateOrUpdatePRCommentCreatesMissingComment(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/wallet/installation":
			writeJSON(t, w, map[string]any{"id": 123})
		case "/app/installations/123/access_tokens":
			writeJSON(t, w, map[string]any{"token": "installation-token"})
		case "/repos/acme/wallet/issues/7/comments":
			if r.Method == http.MethodGet {
				writeJSON(t, w, []map[string]any{})
				return
			}
			require.Equal(t, http.MethodPost, r.Method)
			writeJSON(t, w, map[string]any{"id": 45})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	client := &Client{
		httpClient: server.Client(),
		apiURL:     server.URL,
		clientID:   "client-1",
		privateKey: privateKey,
	}

	result, err := client.CreateOrUpdatePRComment(context.Background(), PRComment{
		Repository:        "acme/wallet",
		PullRequestNumber: 7,
		Marker:            DefaultMarker,
		Body:              "created body",
	})

	require.NoError(t, err)
	require.Equal(t, int64(45), result.CommentID)
}

func TestPullRequestHeadSHA(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/wallet/installation":
			writeJSON(t, w, map[string]any{"id": 123})
		case "/app/installations/123/access_tokens":
			writeJSON(t, w, map[string]any{"token": "installation-token"})
		case "/repos/acme/wallet/pulls/7":
			writeJSON(t, w, map[string]any{"head": map[string]any{"sha": "abc123"}})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	client := &Client{
		httpClient: server.Client(),
		apiURL:     server.URL,
		clientID:   "client-1",
		privateKey: privateKey,
	}

	sha, err := client.PullRequestHeadSHA(context.Background(), "acme/wallet", 7)

	require.NoError(t, err)
	require.Equal(t, "abc123", sha)
}

func TestNewFromEnv(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}))

	t.Setenv("GITHUB_APP_PRIVATE_KEY", strings.ReplaceAll(keyPEM, "\n", `\n`))
	t.Setenv("GITHUB_APP_CLIENT_ID", "client-1")
	t.Setenv("GITHUB_API_URL", "https://github.example/api/")

	client, err := NewFromEnv()

	require.NoError(t, err)
	require.Equal(t, "client-1", client.clientID)
	require.Equal(t, "https://github.example/api", client.apiURL)
	require.NotNil(t, client.privateKey)
}

func TestCreateOrUpdatePRCommentPatchesExplicitCommentID(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/wallet/installation":
			writeJSON(t, w, map[string]any{"id": 123})
		case "/app/installations/123/access_tokens":
			writeJSON(t, w, map[string]any{"token": "installation-token"})
		case "/repos/acme/wallet/issues/comments/99":
			require.Equal(t, http.MethodPatch, r.Method)
			writeJSON(t, w, map[string]any{})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	client := &Client{
		httpClient: server.Client(),
		apiURL:     server.URL,
		clientID:   "client-1",
		privateKey: privateKey,
	}

	result, err := client.CreateOrUpdatePRComment(context.Background(), PRComment{
		Repository:        "acme/wallet",
		PullRequestNumber: 7,
		CommentID:         99,
		Marker:            DefaultMarker,
		Body:              "updated body\n\n" + DefaultMarker,
	})

	require.NoError(t, err)
	require.Equal(t, int64(99), result.CommentID)
}

func TestPRCommentHelpers(t *testing.T) {
	require.Equal(t, DefaultMarker, Marker())
	require.Equal(t, "body\n\n"+DefaultMarker, ensureMarker(" body ", DefaultMarker))
	require.Equal(t, "already "+DefaultMarker, ensureMarker("already "+DefaultMarker, DefaultMarker))

	require.Equal(t, 3, IntFromAny(3))
	require.Equal(t, 4, IntFromAny(int64(4)))
	require.Equal(t, 5, IntFromAny(float64(5)))
	require.Equal(t, 6, IntFromAny(json.Number("6")))
	require.Equal(t, 7, IntFromAny(" 7 "))
	require.Zero(t, IntFromAny("nope"))
	require.Zero(t, IntFromAny(struct{}{}))
}

func writeJSON(t testing.TB, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(value))
}
