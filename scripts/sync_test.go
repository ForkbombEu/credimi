// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDownloadImage(t *testing.T) {
	t.Run("invalid url returns request creation error", func(t *testing.T) {
		_, err := DownloadImage(context.Background(), ":// invalid")
		require.Error(t, err)
		require.ErrorContains(t, err, "create request")
	})

	t.Run("http error status returns error", func(t *testing.T) {
		original := http.DefaultTransport
		t.Cleanup(func() { http.DefaultTransport = original })

		http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("boom")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		})

		_, err := DownloadImage(context.Background(), "https://example.test/image.png")
		require.ErrorContains(t, err, "HTTP 500")
	})

	t.Run("empty body returns error", func(t *testing.T) {
		original := http.DefaultTransport
		t.Cleanup(func() { http.DefaultTransport = original })

		http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		})

		_, err := DownloadImage(context.Background(), "https://example.test/image.png")
		require.ErrorContains(t, err, "empty image")
	})

	t.Run("download failure returns wrapped error", func(t *testing.T) {
		original := http.DefaultTransport
		t.Cleanup(func() { http.DefaultTransport = original })

		http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed")
		})

		_, err := DownloadImage(context.Background(), "https://example.test/image.png")
		require.ErrorContains(t, err, "download failed")
	})

	t.Run("successful download returns file", func(t *testing.T) {
		original := http.DefaultTransport
		t.Cleanup(func() { http.DefaultTransport = original })

		http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("fake-image-bytes")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		})

		file, err := DownloadImage(context.Background(), "https://example.test/path/logo?size=200")
		require.NoError(t, err)
		require.NotNil(t, file)
	})
}

func TestExtractFilenameFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "keeps filename and strips query",
			url:  "https://example.test/static/logo.png?size=200",
			want: "logo.png",
		},
		{
			name: "strips fragment",
			url:  "https://example.test/static/logo.png#section",
			want: "logo.png",
		},
		{
			name: "adds default extension when missing",
			url:  "https://example.test/static/logo",
			want: "logo.jpg",
		},
		{
			name: "falls back to sanitized url when path ends with slash",
			url:  "https://example.test/static/",
			want: "https_example.test_static_.jpg",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, extractFilenameFromURL(tt.url))
		})
	}
}
