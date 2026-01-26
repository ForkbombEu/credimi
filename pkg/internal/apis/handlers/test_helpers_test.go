// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func withMockTransport(t *testing.T, fn roundTripperFunc) {
	oldTransport := http.DefaultTransport
	http.DefaultTransport = fn
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
}

func mockDeeplinkTransport(
	t *testing.T,
	statusCode int,
	responseBody string,
	transportErr error,
) {
	t.Helper()
	withMockTransport(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/api/get-deeplink", req.URL.Path)
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "application/json", req.Header.Get("Content-Type"))

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		var requestData map[string]string
		err = json.Unmarshal(body, &requestData)
		require.NoError(t, err)
		require.Contains(t, requestData, "yaml")

		if transportErr != nil {
			return nil, transportErr
		}

		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(strings.NewReader(responseBody)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})
}
