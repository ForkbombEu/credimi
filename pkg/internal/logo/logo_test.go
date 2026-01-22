// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package logo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data/"

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func withMockTransport(t *testing.T, fn roundTripperFunc) {
	oldTransport := http.DefaultTransport
	http.DefaultTransport = fn
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
}

func setupTestApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(
		testDataDir,
	)
	require.NoError(t, err)
	LogoHooks(app)

	return app
}

func getTestOrgID() (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	filter := `name="userA's organization"`

	record, err := app.FindFirstRecordByFilter("organizations", filter)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func TestLogoHooks_Valid(t *testing.T) {
	withMockTransport(t, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("fake logo image data")),
			Header:     http.Header{"Content-Type": []string{"image/jpeg"}},
		}, nil
	})

	app := setupTestApp(t)
	defer app.Cleanup()
	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	own, _ := getTestOrgID()
	record.Set("owner", own)
	record.Set("logo", "")

	record.Set("logo_url", "https://example.com/logo.jpg")

	require.Empty(
		t,
		record.GetString("logo"),
		"The logo field should be empty before saving",
	)

	err = app.Save(record)
	require.NoError(t, err)

	logoField := record.GetString("logo")
	require.NotEmpty(
		t,
		logoField,
		"The logo field should have been set by LogoHooks after saving",
	)

	require.True(t, strings.HasPrefix(logoField, "logo"), "Filename should start with 'logo'")
	require.True(t, strings.HasSuffix(logoField, ".jpg"), "Filename should end with '.jpg'")

	logokey := record.BaseFilesPath() + "/" + record.GetString("logo")

	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}
	defer fsys.Close()

	r, err := fsys.GetFile(logokey)
	if err != nil {
		t.Fatalf("Failed to get file: %v", err)
	}
	defer r.Close()

	downloadedData, err := io.ReadAll(r)
	require.NoError(t, err)

	require.NotEmpty(t, downloadedData, "The downloaded logo file should not be empty")

	t.Logf("Test complete: LogoHooks worked correctly")
}

func TestLogoHooks_UpdateAddLogoURL(t *testing.T) {
	withMockTransport(t, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("fake logo image data")),
			Header:     http.Header{"Content-Type": []string{"image/jpeg"}},
		}, nil
	})

	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	own, _ := getTestOrgID()
	record.Set("owner", own)
	record.Set("logo", "")
	record.Set("logo_url", "")

	err = app.Save(record)
	require.NoError(t, err)

	require.Empty(t, record.GetString("logo"), "Logo field should be empty initially")

	record.Set("logo_url", "https://example.com/logo.jpg")

	err = app.Save(record)
	require.NoError(t, err)

	logoField := record.GetString("logo")
	require.NotEmpty(t, logoField, "Logo field should have been set by LogoHooks after update")

	require.True(t, strings.HasPrefix(logoField, "logo"), "Filename should start with 'logo'")
	require.True(t, strings.HasSuffix(logoField, ".jpg"), "Filename should end with '.jpg'")

	logokey := record.BaseFilesPath() + "/" + logoField
	fsys, err := app.NewFilesystem()
	require.NoError(t, err)
	defer fsys.Close()

	r, err := fsys.GetFile(logokey)
	require.NoError(t, err)
	defer r.Close()

	downloadedData, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NotEmpty(t, downloadedData, "Downloaded logo file should not be empty")

	t.Logf("Test complete: LogoHooks worked correctly on record update")
}

func TestLogoHooks_InvalidURL(t *testing.T) {
	withMockTransport(t, func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("network error")
	})

	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	own, _ := getTestOrgID()
	record.Set("owner", own)
	record.Set("logo", "")
	record.Set("logo_url", "https://example.com/logo.png")

	err = app.Save(record)
	require.NoError(t, err, "Save should succeed even if logo download fails")

	logoField := record.GetString("logo")

	require.Empty(t, logoField, "Logo field should remain empty when URL is invalid")

	t.Logf("Test complete: LogoHooks handled invalid URL gracefully")
}

func TestLogoHooks_HTTPError(t *testing.T) {
	withMockTransport(t, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
		}, nil
	})

	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	own, _ := getTestOrgID()
	record.Set("owner", own)
	record.Set("logo_url", "https://example.com/notfound.png")

	err = app.Save(record)
	require.NoError(t, err, "Save should succeed even if download fails")

	logoField := record.GetString("logo")
	require.Empty(t, logoField, "Logo field should remain empty when download fails")
}

func TestExtractFilenameFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{
			url:      "https://example.com/logo.png",
			expected: "logo.png",
		},
		{
			url:      "https://example.com/logo.png?width=100",
			expected: "logo.png",
		},
		{
			url:      "https://example.com/path/to/image.jpg",
			expected: "image.jpg",
		},
		{
			url:      "https://example.com/",
			expected: "https_example.com_.jpg",
		},
		{
			url:      "https://example.com/logo",
			expected: "logo.jpg",
		},
		{
			url:      "https://example.com/logo.png#section",
			expected: "logo.png", // Test per fragment
		},
		{
			url:      "https://example.com/logo.png?width=100#section",
			expected: "logo.png", // Test per query + fragment
		},
		{
			url:      "https://example.com/logo#fragment",
			expected: "logo.jpg", // Test per fragment senza estensione
		},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := extractFilenameFromURL(tc.url)
			require.Equal(t, tc.expected, result)
		})
	}
}
func TestLogoHooks_WithUnsavedFiles(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	own, _ := getTestOrgID()
	record.Set("owner", own)
	record.Set("logo_url", "https://example.com/logo.png")

	testFile, err := filesystem.NewFileFromBytes([]byte("manual upload data"), "manual-logo.jpg")
	require.NoError(t, err)

	record.Set("logo", []*filesystem.File{testFile})

	err = app.Save(record)
	require.NoError(t, err, "Save should succeed when there are unsaved files")

	logoField := record.GetString("logo")
	require.NotEmpty(t, logoField, "Logo field should contain the manually uploaded file")

	logokey := record.BaseFilesPath() + "/" + logoField
	fsys, err := app.NewFilesystem()
	require.NoError(t, err)
	defer fsys.Close()

	r, err := fsys.GetFile(logokey)
	require.NoError(t, err)
	defer r.Close()

	fileData, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, []byte("manual upload data"), fileData, "Should preserve manually uploaded file data")

	t.Logf("Test complete: LogoHooks skipped download when unsaved files exist")
}

func TestDownloadImage_InvalidURL(t *testing.T) {
	file, err := DownloadImage(context.Background(), "http://[::1]:namedport/logo.png")

	require.Error(t, err)
	require.Nil(t, file)
	require.True(t, strings.Contains(err.Error(), "create request") || strings.Contains(err.Error(), "download failed"),
		"Error should mention request creation or download: %v", err)
}

func TestDownloadImage_EmptyURL(t *testing.T) {
	file, err := DownloadImage(context.Background(), "")

	require.Error(t, err)
	require.Nil(t, file)
	require.Contains(t, err.Error(), "unsupported protocol scheme", "Error should mention protocol scheme for empty URL")
}

func TestDownloadImage_MalformedURL(t *testing.T) {
	file, err := DownloadImage(context.Background(), "ftp://example.com/logo.png")

	require.Error(t, err)
	require.Nil(t, file)
	require.Contains(t, err.Error(), "unsupported protocol scheme", "Error should mention protocol scheme")
}

func TestDownloadImage_ContextInRequest(t *testing.T) {
	withMockTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.Context().Value("testKey") != "testValue" {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader("missing context")),
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("test image")),
			Header:     http.Header{"Content-Type": []string{"image/jpeg"}},
		}, nil
	})

	ctx := context.WithValue(context.Background(), "testKey", "testValue")
	file, err := DownloadImage(ctx, "https://example.com/test.jpg")

	require.NoError(t, err)
	require.NotNil(t, file)
}

func TestDownloadImage_EmptyImage(t *testing.T) {
	withMockTransport(t, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{"Content-Type": []string{"image/jpeg"}},
		}, nil
	})

	file, err := DownloadImage(context.Background(), "https://example.com/empty.jpg")

	require.Error(t, err)
	require.Nil(t, file)
	require.Contains(t, err.Error(), "empty image", "Error should mention empty image")
}
