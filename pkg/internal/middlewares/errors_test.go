// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"unsafe"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

// errorMiddlewareResponse decodes the body written by ErrorHandlingMiddleware.
type errorMiddlewareResponse = apierror.Response

func setNext(e *core.RequestEvent, fn func() error) {
	eventField := reflect.ValueOf(e).Elem().FieldByName("Event")
	hookEvent := eventField.FieldByName("Event")
	nextField := hookEvent.FieldByName("next")
	reflect.NewAt(nextField.Type(), unsafe.Pointer(nextField.UnsafeAddr())).Elem().
		Set(reflect.ValueOf(fn))
}

func TestErrorHandlingMiddleware(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   map[string]any
	}{
		{
			name:       "credimi api error",
			err:        apierror.New(http.StatusBadRequest, "yaml", "invalid yaml", "line 3"),
			wantStatus: http.StatusBadRequest,
			wantBody: map[string]any{
				"apiVersion": "2.0",
				"message":    "line 3",
				"error": map[string]any{
					"code": float64(http.StatusBadRequest), "domain": "yaml",
					"reason": "invalid yaml", "message": "line 3",
				},
			},
		},
		{
			name: "wrapped credimi api error",
			err: fmt.Errorf(
				"start: %w",
				apierror.New(http.StatusConflict, "ticket", "already queued", "t-1"),
			),
			wantStatus: http.StatusConflict,
			wantBody: map[string]any{
				"apiVersion": "2.0",
				"message":    "t-1",
				"error": map[string]any{
					"code": float64(http.StatusConflict), "domain": "ticket",
					"reason": "already queued", "message": "t-1",
				},
			},
		},
		{
			name:       "pocketbase error keeps its status",
			err:        apis.NewForbiddenError("not your record", nil),
			wantStatus: http.StatusForbidden,
			wantBody: map[string]any{
				"apiVersion": "2.0",
				"message":    "Not your record.",
				"error": map[string]any{
					"code": float64(http.StatusForbidden), "domain": "request",
					"reason": "Forbidden", "message": "Not your record.",
				},
			},
		},
		{
			name:       "unhandled error",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantBody: map[string]any{
				"apiVersion": "2.0",
				"message":    "Internal Server Error",
				"error": map[string]any{
					"code": float64(http.StatusInternalServerError), "domain": "internal",
					"reason": "UnhandledException", "message": "boom",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e := &core.RequestEvent{
				Event: router.Event{
					Request:  httptest.NewRequest(http.MethodGet, "/", nil),
					Response: rec,
				},
			}
			setNext(e, func() error { return tc.err })

			require.NoError(t, ErrorHandlingMiddleware(e))
			require.Equal(t, tc.wantStatus, rec.Code)

			var body map[string]any
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
			require.Equal(t, tc.wantBody, body)
		})
	}
}

func TestErrorHandlingMiddlewarePassesSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{
		Event: router.Event{
			Request:  httptest.NewRequest(http.MethodGet, "/", nil),
			Response: rec,
		},
	}
	setNext(e, func() error { return nil })

	require.NoError(t, ErrorHandlingMiddleware(e))
	require.Zero(t, rec.Body.Len())
}
