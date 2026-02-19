// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
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
	require.Contains(t, rec.Body.String(), "\"apiVersion\":\"2.0\"")
	require.Contains(t, rec.Body.String(), "\"domain\":\"domain\"")
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
	require.Contains(t, rec.Body.String(), "UnhandledException")
}
