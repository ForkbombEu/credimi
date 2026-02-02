// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apierror

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

type apiErrorResponse struct {
	Status  int    `json:"status"`
	Domain  string `json:"error"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

func TestAPIError_Error(t *testing.T) {
	apiErr := New(http.StatusBadRequest, "test-domain", "bad", "something went wrong")

	require.Equal(t, "[test-domain:bad] something went wrong", apiErr.Error())
}

func TestAPIError_JSON(t *testing.T) {
	apiErr := New(http.StatusBadRequest, "test-domain", "bad", "something went wrong")

	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{Event: router.Event{Request: req, Response: rec}}

	err := apiErr.JSON(event)
	require.NoError(t, err)

	res := rec.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusBadRequest, res.StatusCode)

	var body apiErrorResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	require.Equal(t, http.StatusBadRequest, body.Status)
	require.Equal(t, "test-domain", body.Domain)
	require.Equal(t, "bad", body.Reason)
	require.Equal(t, "something went wrong", body.Message)
}
