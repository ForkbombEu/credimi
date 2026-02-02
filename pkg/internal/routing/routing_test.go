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
