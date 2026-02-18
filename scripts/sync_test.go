// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractFilenameFromURL(t *testing.T) {
	require.Equal(t, "logo.png", extractFilenameFromURL("https://example.test/logo.png"))
	require.Equal(
		t,
		"logo.jpg",
		extractFilenameFromURL("https://example.test/logo"),
	)
	require.Equal(
		t,
		"logo.png",
		extractFilenameFromURL("https://example.test/logo.png?size=small"),
	)
	require.Equal(
		t,
		"logo.png",
		extractFilenameFromURL("https://example.test/logo.png#section"),
	)
	require.Equal(
		t,
		"https_example.test_path_.jpg",
		extractFilenameFromURL("https://example.test/path/"),
	)
}

func TestDownloadImageSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "image-bytes")
	}))
	t.Cleanup(server.Close)

	file, err := DownloadImage(context.Background(), server.URL+"/logo.png")
	require.NoError(t, err)
	require.NotNil(t, file)
	require.Equal(t, "logo.png", file.OriginalName)
	require.NotZero(t, file.Size)
}

func TestDownloadImageHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	_, err := DownloadImage(context.Background(), server.URL+"/logo.png")
	require.Error(t, err)
}

func TestDownloadImageEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	_, err := DownloadImage(context.Background(), server.URL+"/logo.png")
	require.Error(t, err)
}
