// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package routing

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

type samplePayload struct {
	Name string `json:"name" validate:"required"`
}

func TestGetValidatedInput(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	defer app.Cleanup()

	t.Run("nil value", func(t *testing.T) {
		ctx := context.Background()
		rec := newRequestEvent(app, http.MethodGet, "/", nil, ctx)

		val, err := GetValidatedInput[samplePayload](rec.Event)
		require.NoError(t, err)
		require.Equal(t, samplePayload{}, val)
	})

	t.Run("correct type", func(t *testing.T) {
		payload := samplePayload{Name: "ok"}
		ctx := context.WithValue(context.Background(), middlewares.ValidatedInputKey, payload)
		rec := newRequestEvent(app, http.MethodGet, "/", nil, ctx)

		val, err := GetValidatedInput[samplePayload](rec.Event)
		require.NoError(t, err)
		require.Equal(t, payload, val)
	})

	t.Run("type mismatch", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), middlewares.ValidatedInputKey, "bad")
		rec := newRequestEvent(app, http.MethodGet, "/", nil, ctx)

		val, err := GetValidatedInput[samplePayload](rec.Event)
		require.NoError(t, err)
		require.Equal(t, samplePayload{}, val)
		require.Equal(t, http.StatusInternalServerError, rec.Recorder.Code)
	})
}

func TestRegisterRoutesWithValidation(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	defer app.Cleanup()

	var handlerCalled atomic.Bool
	var middlewareCalled atomic.Bool

	customMiddleware := &hook.Handler[*core.RequestEvent]{
		Id: "custom",
		Func: func(e *core.RequestEvent) error {
			middlewareCalled.Store(true)
			return e.Next()
		},
	}

	route := RouteDefinition{
		Method:        http.MethodPost,
		Path:          "/validated",
		RequestSchema: samplePayload{},
		Handler: func() func(*core.RequestEvent) error {
			return func(e *core.RequestEvent) error {
				handlerCalled.Store(true)
				return e.String(http.StatusOK, "ok")
			}
		},
		Middlewares:         []*hook.Handler[*core.RequestEvent]{customMiddleware},
		ExcludedMiddlewares: []string{"custom"},
	}

	r := router.NewRouter(
		func(w http.ResponseWriter, req *http.Request) (*core.RequestEvent, router.EventCleanupFunc) {
			return &core.RequestEvent{
				App:   app,
				Event: router.Event{Response: w, Request: req},
			}, nil
		},
	)

	RegisterRoutesWithValidation(app, r.RouterGroup, []RouteDefinition{route}, false)
	mux, err := r.BuildMux()
	require.NoError(t, err)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/validated", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)
	require.Equal(t, http.StatusBadRequest, res.Code)
	require.False(t, handlerCalled.Load())
	require.False(t, middlewareCalled.Load())
}

func TestRegisterRoutesWithValidation_RequireAuth(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	defer app.Cleanup()

	var handlerCalled atomic.Bool
	secureRoute := RouteDefinition{
		Method: http.MethodGet,
		Path:   "/secure",
		Handler: func() func(*core.RequestEvent) error {
			return func(e *core.RequestEvent) error {
				handlerCalled.Store(true)
				return e.String(http.StatusOK, "ok")
			}
		},
	}

	r := router.NewRouter(
		func(w http.ResponseWriter, req *http.Request) (*core.RequestEvent, router.EventCleanupFunc) {
			return &core.RequestEvent{
				App:   app,
				Event: router.Event{Response: w, Request: req},
			}, nil
		},
	)

	RegisterRoutesWithValidation(app, r.RouterGroup, []RouteDefinition{secureRoute}, true)
	mux, err := r.BuildMux()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)
	require.Equal(t, http.StatusUnauthorized, res.Code)
	require.False(t, handlerCalled.Load())
}

func TestRegisterRoutesWithValidation_MethodCoverage(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	defer app.Cleanup()

	var putCalled atomic.Bool
	var patchCalled atomic.Bool
	var deleteCalled atomic.Bool
	var getCalled atomic.Bool
	var headCalled atomic.Bool
	var optionsCalled atomic.Bool
	var unsupportedCalled atomic.Bool

	routes := []RouteDefinition{
		{
			Method:        http.MethodPut,
			Path:          "/put",
			RequestSchema: samplePayload{},
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					putCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method:        http.MethodPatch,
			Path:          "/patch",
			RequestSchema: samplePayload{},
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					patchCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method:        http.MethodDelete,
			Path:          "/delete",
			RequestSchema: samplePayload{},
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					deleteCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/get",
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					getCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method: http.MethodHead,
			Path:   "/head",
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					headCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method: http.MethodOptions,
			Path:   "/options",
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					optionsCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method: "TRACE",
			Path:   "/unsupported",
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					unsupportedCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
	}

	r := router.NewRouter(
		func(w http.ResponseWriter, req *http.Request) (*core.RequestEvent, router.EventCleanupFunc) {
			return &core.RequestEvent{
				App:   app,
				Event: router.Event{Response: w, Request: req},
			}, nil
		},
	)

	RegisterRoutesWithValidation(app, r.RouterGroup, routes, false)
	mux, err := r.BuildMux()
	require.NoError(t, err)

	for _, path := range []string{"/put", "/patch", "/delete"} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`))
		req.Method = map[string]string{
			"/put":    http.MethodPut,
			"/patch":  http.MethodPatch,
			"/delete": http.MethodDelete,
		}[path]
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		require.Equal(t, http.StatusBadRequest, res.Code)
	}

	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/get", nil))
	require.Equal(t, http.StatusNoContent, res.Code)
	require.True(t, getCalled.Load())

	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodHead, "/head", nil))
	require.Equal(t, http.StatusNoContent, res.Code)
	require.True(t, headCalled.Load())

	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodOptions, "/options", nil))
	require.Equal(t, http.StatusNoContent, res.Code)
	require.True(t, optionsCalled.Load())

	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodTrace, "/unsupported", nil))
	require.Equal(t, http.StatusNotFound, res.Code)
	require.False(t, unsupportedCalled.Load())

	require.False(t, putCalled.Load())
	require.False(t, patchCalled.Load())
	require.False(t, deleteCalled.Load())
}

func TestRegisterRoutesWithoutValidation_MethodCoverage(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	defer app.Cleanup()

	var postCalled atomic.Bool
	var getCalled atomic.Bool
	var putCalled atomic.Bool
	var patchCalled atomic.Bool
	var deleteCalled atomic.Bool
	var unsupportedCalled atomic.Bool

	routes := []RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/post",
			RequestSchema: samplePayload{},
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					postCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/get",
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					getCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method:        http.MethodPut,
			Path:          "/put",
			RequestSchema: samplePayload{},
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					putCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method:        http.MethodPatch,
			Path:          "/patch",
			RequestSchema: samplePayload{},
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					patchCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method:        http.MethodDelete,
			Path:          "/delete",
			RequestSchema: samplePayload{},
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					deleteCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
		{
			Method: "TRACE",
			Path:   "/unsupported",
			Handler: func() func(*core.RequestEvent) error {
				return func(e *core.RequestEvent) error {
					unsupportedCalled.Store(true)
					return e.NoContent(http.StatusNoContent)
				}
			},
		},
	}

	r := router.NewRouter(
		func(w http.ResponseWriter, req *http.Request) (*core.RequestEvent, router.EventCleanupFunc) {
			return &core.RequestEvent{
				App:   app,
				Event: router.Event{Response: w, Request: req},
			}, nil
		},
	)

	RegisterRoutesWithoutValidation(app, r.RouterGroup, routes)
	mux, err := r.BuildMux()
	require.NoError(t, err)

	requests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/post"},
		{method: http.MethodGet, path: "/get"},
		{method: http.MethodPut, path: "/put"},
		{method: http.MethodPatch, path: "/patch"},
		{method: http.MethodDelete, path: "/delete"},
	}

	for _, tc := range requests {
		body := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest(tc.method, tc.path, body)
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		require.Equalf(t, http.StatusNoContent, res.Code, "unexpected status for %s %s", tc.method, tc.path)
	}

	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodTrace, "/unsupported", nil))
	require.Equal(t, http.StatusNotFound, res.Code)
	require.False(t, unsupportedCalled.Load())

	require.True(t, postCalled.Load())
	require.True(t, getCalled.Load())
	require.True(t, putCalled.Load())
	require.True(t, patchCalled.Load())
	require.True(t, deleteCalled.Load())
}

type requestEventRecorder struct {
	Event    *core.RequestEvent
	Recorder *httptest.ResponseRecorder
}

func newRequestEvent(
	app core.App,
	method, path string,
	body *bytes.Buffer,
	ctx context.Context,
) requestEventRecorder {
	if body == nil {
		body = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, body)
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	rec := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Event: router.Event{Response: rec, Request: req}}

	return requestEventRecorder{Event: event, Recorder: rec}
}
