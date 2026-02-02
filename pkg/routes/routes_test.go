// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package routes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func TestCreateReverseProxy(t *testing.T) {
	var received *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	targetURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	handler := createReverseProxy(server.URL)

	req := httptest.NewRequest(http.MethodGet, "http://example.test/foo", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	req.Header.Set("Origin", "https://origin.example")
	req.Header.Set("Referer", "https://referer.example/path")

	rec := httptest.NewRecorder()
	event := &core.RequestEvent{Event: router.Event{Request: req, Response: rec}}

	err = handler(event)
	require.NoError(t, err)
	require.NotNil(t, received)

	require.Equal(t, "/foo", received.URL.Path)
	require.Equal(t, targetURL.Host, received.Host)
	require.Contains(t, received.Header.Get("X-Forwarded-For"), "1.2.3.4:5678")
	require.Equal(t, "https://origin.example", received.Header.Get("Origin"))
	require.Equal(t, "https://referer.example/path", received.Header.Get("Referer"))
}

func TestCreateReverseProxy_InvalidTarget(t *testing.T) {
	handler := createReverseProxy("://bad-url")

	req := httptest.NewRequest(http.MethodGet, "http://example.test/foo", nil)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{Event: router.Event{Request: req, Response: rec}}

	err := handler(event)
	require.Error(t, err)
}
