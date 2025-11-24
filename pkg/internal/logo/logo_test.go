// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package logo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data/"

func setupTestApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(
		testDataDir,
	) // qui la metti che punta a test_pb_data con il path relativo rispetto al tuo test file
	require.NoError(t, err)
	LogoHooks(app)

	return app
}

func getOrgIDfromName(collectionNameOrID string, name string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	filter := fmt.Sprintf(`name="%s"`, name)

	record, err := app.FindFirstRecordByFilter(collectionNameOrID, filter)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func TestLogoHooks_Valid(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()
	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	own, _ := getOrgIDfromName("organizations", "userA's organization")
	record.Set("owner", own)
	record.Set("logo", "")

	record.Set(
		"logo_url",
		"https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/25/de/dc/25dedc8b-0e66-70cd-1530-082a007ef642/AppIcon-0-0-1x_U007ephone-0-1-85-220.png/100x100bb.jpg",
	)

	require.Empty(
		t,
		record.GetString("logo"),
		"The logo field should have been set by LogoHooks after saving",
	)

	err = app.Save(record)
	require.NoError(t, err)

	logoField := record.GetString("logo")
	require.NotEmpty(
		t,
		logoField,
		"The logo field shouldn't have been set by LogoHooks after saving",
	)

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
	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	own, _ := getOrgIDfromName("organizations", "userA's organization")
	record.Set("owner", own)
	record.Set("logo", "")
	record.Set("logo_url", "")

	err = app.Save(record)
	require.NoError(t, err)

	require.Empty(t, record.GetString("logo"), "Logo field should be empty initially")

	record.Set(
		"logo_url",
		"https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/25/de/dc/25dedc8b-0e66-70cd-1530-082a007ef642/AppIcon-0-0-1x_U007ephone-0-1-85-220.png/100x100bb.jpg",
	)

	err = app.Save(record)
	require.NoError(t, err)

	logoField := record.GetString("logo")
	require.NotEmpty(t, logoField, "Logo field should have been set by LogoHooks after update")

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
	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	own, _ := getOrgIDfromName("organizations", "userA's organization")
	record.Set("owner", own)
	record.Set("logo", "")
	record.Set("logo_url", "https://invalid-domain-that-does-not-exist-12345.com/logo.png")

	err = app.Save(record)
	require.NoError(t, err, "Save should succeed even if logo download fails")

	logoField := record.GetString("logo")

	require.Empty(t, logoField, "Logo field should remain empty when URL is invalid")

	t.Logf("Test complete: LogoHooks handled invalid URL gracefully")
}

func TestDownloadWithCustomClient_HTTPStatusErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		statusText string
		expectErr  string
	}{
		{
			name:       "Status 400 Bad Request",
			statusCode: http.StatusBadRequest,
			statusText: "Bad Request",
			expectErr:  "HTTP 400: 400 Bad Request",
		},
		{
			name:       "Status 403 Forbidden",
			statusCode: http.StatusForbidden,
			statusText: "Forbidden",
			expectErr:  "HTTP 403: 403 Forbidden",
		},
		{
			name:       "Status 404 Not Found",
			statusCode: http.StatusNotFound,
			statusText: "Not Found",
			expectErr:  "HTTP 404: 404 Not Found",
		},
		{
			name:       "Status 500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			statusText: "Internal Server Error",
			expectErr:  "HTTP 500: 500 Internal Server Error",
		},
		{
			name:       "Status 503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			statusText: "Service Unavailable",
			expectErr:  "HTTP 503: 503 Service Unavailable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				if tc.statusText != "" {
					w.Write([]byte(tc.statusText))
				}
			}))
			defer server.Close()

			file, err := downloadWithCustomClient(context.Background(), server.URL)

			require.Error(t, err)
			assert.Nil(t, file)
			assert.Contains(t, err.Error(), tc.expectErr)
		})
	}
}

func TestDownloadWithCustomClient_EmptyResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "image/jpeg")
	}))
	defer server.Close()

	file, err := downloadWithCustomClient(context.Background(), server.URL)

	require.Error(t, err)
	assert.Nil(t, file)
	assert.Equal(t, "empty response", err.Error())
}

func TestDownloadWithCustomClient_ContentTypeDetection(t *testing.T) {
	testCases := []struct {
		name           string
		contentType    string
		expectedExt    string
		expectedPrefix string
	}{
		{
			name:           "PNG content type",
			contentType:    "image/png",
			expectedExt:    ".png",
			expectedPrefix: "logo",
		},
		{
			name:           "JPEG content type",
			contentType:    "image/jpeg",
			expectedExt:    ".jpg",
			expectedPrefix: "logo",
		},
		{
			name:           "GIF content type",
			contentType:    "image/gif",
			expectedExt:    ".gif",
			expectedPrefix: "logo",
		},
		{
			name:           "WebP content type",
			contentType:    "image/webp",
			expectedExt:    ".webp",
			expectedPrefix: "logo",
		},
		{
			name:           "Unknown image type",
			contentType:    "image/unknown",
			expectedExt:    ".jpg",
			expectedPrefix: "logo",
		},
		{
			name:           "No content type header",
			contentType:    "",
			expectedExt:    ".jpg",
			expectedPrefix: "logo",
		},
		{
			name:           "Content type with charset",
			contentType:    "image/png; charset=utf-8",
			expectedExt:    ".png",
			expectedPrefix: "logo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testImageData := []byte{0x01, 0x02, 0x03, 0x04}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.contentType != "" {
					w.Header().Set("Content-Type", tc.contentType)
				}
				w.WriteHeader(http.StatusOK)
				w.Write(testImageData)
			}))
			defer server.Close()

			file, err := downloadWithCustomClient(context.Background(), server.URL)

			require.NoError(t, err)
			require.NotNil(t, file)

			assert.True(t, strings.HasPrefix(file.Name, tc.expectedPrefix),
				"Expected file name to start with '%s', but got '%s'", tc.expectedPrefix, file.Name)
			assert.True(t, strings.HasSuffix(file.Name, tc.expectedExt),
				"Expected file name to end with '%s', but got '%s'", tc.expectedExt, file.Name)

			nameWithoutExt := strings.TrimSuffix(file.Name, tc.expectedExt)
			assert.True(t, len(nameWithoutExt) > len(tc.expectedPrefix),
				"Expected random suffix in file name, but got '%s'", file.Name)
		})
	}
}

func TestFilesystemNewFileFromBytesBehavior(t *testing.T) {
	testData := []byte{0x01, 0x02, 0x03}

	generatedNames := make(map[string]bool)
	for i := 0; i < 10; i++ {
		file, err := filesystem.NewFileFromBytes(testData, "logo.jpg")
		require.NoError(t, err)

		assert.True(t, strings.HasPrefix(file.Name, "logo"))
		assert.True(t, strings.HasSuffix(file.Name, ".jpg"))

		if generatedNames[file.Name] {
			t.Errorf("Duplicate filename generated: %s", file.Name)
		}
		generatedNames[file.Name] = true
	}

	assert.Equal(t, 10, len(generatedNames), "Should generate 10 unique filenames")
}
