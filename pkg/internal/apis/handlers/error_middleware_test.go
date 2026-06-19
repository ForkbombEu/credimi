// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"unsafe"

	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

type testErrorResponse struct {
	APIVersion string `json:"apiVersion"`
	Message    string `json:"message"`
	Error      struct {
		Code    int    `json:"code"`
		Domain  string `json:"domain"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	} `json:"error"`
}

func requireHandlerErrorHandled(t testing.TB, rec *httptest.ResponseRecorder, err error) {
	t.Helper()

	if err == nil {
		return
	}

	e := &core.RequestEvent{
		Event: router.Event{
			Request:  httptest.NewRequest(http.MethodGet, "/", nil),
			Response: rec,
		},
	}
	setErrorMiddlewareNext(e, func() error { return err })
	require.NoError(t, middlewares.ErrorHandlingMiddleware(e))
}

func decodeHandlerErrorResponse(t testing.TB, rec *httptest.ResponseRecorder) testErrorResponse {
	t.Helper()

	var body testErrorResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	return body
}

func setErrorMiddlewareNext(e *core.RequestEvent, fn func() error) {
	eventField := reflect.ValueOf(e).Elem().FieldByName("Event")
	hookEvent := eventField.FieldByName("Event")
	nextField := hookEvent.FieldByName("next")
	reflect.NewAt(nextField.Type(), unsafe.Pointer(nextField.UnsafeAddr())).Elem().
		Set(reflect.ValueOf(fn))
}
