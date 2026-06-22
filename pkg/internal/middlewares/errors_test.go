// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"unsafe"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

type errorMiddlewareResponse struct {
	APIVersion string `json:"apiVersion"`
	Message    string `json:"message"`
	Error      struct {
		Code    int    `json:"code"`
		Domain  string `json:"domain"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	} `json:"error"`
}

func setNext(e *core.RequestEvent, fn func() error) {
	eventField := reflect.ValueOf(e).Elem().FieldByName("Event")
	hookEvent := eventField.FieldByName("Event")
	nextField := hookEvent.FieldByName("next")
	reflect.NewAt(nextField.Type(), unsafe.Pointer(nextField.UnsafeAddr())).Elem().
		Set(reflect.ValueOf(fn))
}

func TestErrorHandlingMiddlewareAPIError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	e := &core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}
	setNext(e, func() error {
		return apierror.New(http.StatusBadRequest, "domain", "message", "reason")
	})

	err := ErrorHandlingMiddleware(e)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body errorMiddlewareResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Equal(t, "2.0", body.APIVersion)
	require.Equal(t, "reason", body.Message)
	require.Equal(t, http.StatusBadRequest, body.Error.Code)
	require.Equal(t, "domain", body.Error.Domain)
	require.Equal(t, "message", body.Error.Reason)
	require.Equal(t, "reason", body.Error.Message)
}

func TestErrorHandlingMiddlewareUnhandledError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	e := &core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}
	setNext(e, func() error {
		return errors.New("boom")
	})

	err := ErrorHandlingMiddleware(e)
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var body errorMiddlewareResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Equal(t, "2.0", body.APIVersion)
	require.Equal(t, "Internal Server Error", body.Message)
	require.Equal(t, http.StatusInternalServerError, body.Error.Code)
	require.Equal(t, "internal", body.Error.Domain)
	require.Equal(t, "UnhandledException", body.Error.Reason)
	require.Equal(t, "boom", body.Error.Message)
}
