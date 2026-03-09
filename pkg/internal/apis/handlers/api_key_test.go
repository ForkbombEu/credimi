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
