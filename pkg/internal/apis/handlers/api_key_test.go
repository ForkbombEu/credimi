// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

type tokenStub struct {
	token string
	err   error
}

func (t tokenStub) NewAuthToken() (string, error) {
	if t.err != nil {
		return "", t.err
	}
	return t.token, nil
}

type apiKeyGenerationServiceStub struct {
	userKey              string
	internalAdminKey     string
	userErr              error
	internalAdminErr     error
	userCalls            int
	internalAdminCalls   int
	lastUserID           string
	lastSuperuserID      string
	lastGeneratedKeyName string
}

func (s *apiKeyGenerationServiceStub) GenerateApiKey(userID, name string) (string, error) {
	s.userCalls++
	s.lastUserID = userID
	s.lastGeneratedKeyName = name
	if s.userErr != nil {
		return "", s.userErr
	}
	return s.userKey, nil
}

func (s *apiKeyGenerationServiceStub) GenerateInternalAdminAPIKey(
	superuserID, name string,
) (string, error) {
	s.internalAdminCalls++
	s.lastSuperuserID = superuserID
	s.lastGeneratedKeyName = name
	if s.internalAdminErr != nil {
		return "", s.internalAdminErr
	}
	return s.internalAdminKey, nil
}

func TestGenerateAuthenticateApiKeyResponse(t *testing.T) {
	response, err := generateAuthenticateApiKeyResponse(tokenStub{token: "token-1"})
	require.NoError(t, err)
	require.Equal(t, "token-1", response.Token)

	_, err = generateAuthenticateApiKeyResponse(tokenStub{err: errors.New("boom")})
	require.Error(t, err)
}

func TestGenerateApiKeyInvalidInput(t *testing.T) {
	handler := GenerateApiKey()
	req := httptest.NewRequest(http.MethodPost, "/api/apikey/generate", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	auth := core.NewRecord(core.NewAuthCollection("users"))
	auth.Id = "user-1"

	e := &core.RequestEvent{
		Auth: auth,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	err := handler(e)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "name_required")
}

func TestGenerateAPIKeyForPrincipalUser(t *testing.T) {
	stub := &apiKeyGenerationServiceStub{userKey: "user-key"}
	auth := core.NewRecord(core.NewAuthCollection("users"))
	auth.Id = "user-1"

	key, err := generateAPIKeyForPrincipal(stub, auth, "key-name")
	require.NoError(t, err)
	require.Equal(t, "user-key", key)
	require.Equal(t, 1, stub.userCalls)
	require.Equal(t, 0, stub.internalAdminCalls)
	require.Equal(t, "user-1", stub.lastUserID)
	require.Equal(t, "key-name", stub.lastGeneratedKeyName)
}

func TestGenerateAPIKeyForPrincipalSuperuser(t *testing.T) {
	stub := &apiKeyGenerationServiceStub{internalAdminKey: "internal-key"}
	auth := core.NewRecord(core.NewAuthCollection("_superusers"))
	auth.Id = "superuser-1"

	key, err := generateAPIKeyForPrincipal(stub, auth, "key-name")
	require.NoError(t, err)
	require.Equal(t, "internal-key", key)
	require.Equal(t, 0, stub.userCalls)
	require.Equal(t, 1, stub.internalAdminCalls)
	require.Equal(t, "superuser-1", stub.lastSuperuserID)
	require.Equal(t, "key-name", stub.lastGeneratedKeyName)
}

func TestGenerateAPIKeyForPrincipalUnsupportedCollection(t *testing.T) {
	stub := &apiKeyGenerationServiceStub{}
	auth := core.NewRecord(core.NewAuthCollection("admins"))
	auth.Id = "admin-1"

	_, err := generateAPIKeyForPrincipal(stub, auth, "key-name")
	require.Error(t, err)

	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusForbidden, apiErr.Code)
	require.Equal(t, "unsupported_auth_collection", apiErr.Reason)
	require.Equal(t, 0, stub.userCalls)
	require.Equal(t, 0, stub.internalAdminCalls)
}
