// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package logo

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/safehttp"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data/"

func setupTestApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(
		testDataDir,
	)
	require.NoError(t, err)
	LogoHooks(app)

	return app
}

// setupTestAppAllowingLoopback binds the hooks with a client that may reach
// httptest servers, which listen on loopback.
func setupTestAppAllowingLoopback(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	bindLogoHooks(app, newLogoHTTPClient(allowAllIPs))

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
	// Crea un server di test per simulare il download
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(testPNG(t))
	}))
	defer ts.Close()

	app := setupTestAppAllowingLoopback(t)
	defer app.Cleanup()
	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	own, _ := getTestOrgID()
	record.Set("owner", own)
	record.Set("logo", "")

	record.Set("logo_url", ts.URL+"/logo.jpg")

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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(testPNG(t))
	}))
	defer ts.Close()

	app := setupTestAppAllowingLoopback(t)
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

	record.Set("logo_url", ts.URL+"/logo.jpg")

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
	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	own, _ := getTestOrgID()
	record.Set("owner", own)
	record.Set("logo", "")
	record.Set("logo_url", "https://invalid-domain-that-does-not-exist-12345.com/logo.png")

	err = app.Save(record)
	require.NoError(t, err, "Save should succeed even if logo download fails")

	logoField := record.GetString("logo")

	require.Empty(t, logoField, "Logo field should remain empty when URL is invalid")

	t.Logf("Test complete: LogoHooks handled invalid URL gracefully")
}

func TestLogoHooks_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	app := setupTestAppAllowingLoopback(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	own, _ := getTestOrgID()
	record.Set("owner", own)
	record.Set("logo_url", ts.URL+"/notfound.png")

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

	manualUpload := testPNG(t)
	testFile, err := filesystem.NewFileFromBytes(manualUpload, "manual-logo.png")
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
	require.Equal(t, manualUpload, fileData, "Should preserve manually uploaded file data")

	t.Logf("Test complete: LogoHooks skipped download when unsaved files exist")
}

func TestDownloadImage_InvalidURL(t *testing.T) {
	file, err := DownloadImage(context.Background(), "http://[::1]:namedport/logo.png")

	require.Error(t, err)
	require.Nil(t, file)
	require.True(
		t,
		strings.Contains(err.Error(), "create request") ||
			strings.Contains(err.Error(), "download failed"),
		"Error should mention request creation or download: %v",
		err,
	)
}

func TestDownloadImage_EmptyURL(t *testing.T) {
	file, err := DownloadImage(context.Background(), "")

	require.Error(t, err)
	require.Nil(t, file)
	require.Contains(
		t,
		err.Error(),
		"unsupported protocol scheme",
		"Error should mention protocol scheme for empty URL",
	)
}

func TestDownloadImage_MalformedURL(t *testing.T) {
	file, err := DownloadImage(context.Background(), "ftp://example.com/logo.png")

	require.Error(t, err)
	require.Nil(t, file)
	require.Contains(
		t,
		err.Error(),
		"unsupported protocol scheme",
		"Error should mention protocol scheme",
	)
}

func TestDownloadImage_RejectsLoopbackDestination(t *testing.T) {
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Write([]byte("DUMMY-INTERNAL-METADATA-TOKEN"))
	}))
	defer ts.Close()

	file, err := DownloadImage(context.Background(), ts.URL+"/latest/meta-data")

	require.ErrorIs(t, err, safehttp.ErrBlockedDestination)
	require.Nil(t, file)
	require.Zero(t, hits.Load(), "the internal server must never be contacted")
}

func TestLogoHooks_LoopbackLogoURLNotStored(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(testPNG(t))
	}))
	defer ts.Close()

	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	own, err := getTestOrgID()
	require.NoError(t, err)
	record.Set("owner", own)
	record.Set("logo_url", ts.URL+"/latest/meta-data")

	require.NoError(t, app.Save(record), "save must succeed when the logo is refused")
	require.Empty(t, record.GetString("logo"), "a logo from a loopback host must not be stored")
}

func testPNG(t testing.TB) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	return buf.Bytes()
}

func allowAllIPs(net.IP) bool { return true }

func TestDownloadImage_RejectsLocalHostname(t *testing.T) {
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Write(testPNG(t))
	}))
	defer ts.Close()

	_, port, err := net.SplitHostPort(ts.Listener.Addr().String())
	require.NoError(t, err)

	file, err := DownloadImage(context.Background(), "http://localhost:"+port+"/logo.png")

	require.ErrorIs(t, err, safehttp.ErrBlockedDestination)
	require.Nil(t, file)
	require.Zero(t, hits.Load())
}

func TestDownloadImage_RejectsRedirectToBlockedAddress(t *testing.T) {
	var internalHits atomic.Int32
	internal := httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			internalHits.Add(1)
			w.Write(testPNG(t))
		}),
	)
	listener, err := net.Listen("tcp", "127.0.0.2:0")
	if err != nil {
		t.Skipf("127.0.0.2 not bindable: %v", err)
	}
	internal.Listener = listener
	internal.Start()
	defer internal.Close()

	public := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internal.URL+"/latest/meta-data", http.StatusFound)
	}))
	defer public.Close()

	onlyPublicServer := func(ip net.IP) bool { return ip.Equal(net.IPv4(127, 0, 0, 1)) }
	file, err := downloadImage(
		context.Background(),
		newLogoHTTPClient(onlyPublicServer),
		public.URL+"/logo.png",
	)

	require.ErrorIs(t, err, safehttp.ErrBlockedDestination)
	require.Nil(t, file)
	require.Zero(t, internalHits.Load())
}

func TestDownloadImage_ValidatesBody(t *testing.T) {
	pngData := testPNG(t)
	exactlyMax := append(append([]byte{}, pngData...), make([]byte, maxLogoSize-len(pngData))...)

	tests := []struct {
		name    string
		body    []byte
		wantErr string
	}{
		{name: "png accepted", body: pngData},
		{name: "image at size cap accepted", body: exactlyMax},
		{name: "image over size cap rejected", body: append(exactlyMax, 0), wantErr: "exceeds"},
		{
			name:    "non image rejected",
			body:    []byte("DUMMY-INTERNAL-METADATA-TOKEN"),
			wantErr: "unsupported image content type",
		},
		{
			name:    "html rejected",
			body:    []byte("<html><script>alert(1)</script></html>"),
			wantErr: "unsupported image content type",
		},
		{name: "empty rejected", body: nil, wantErr: "empty image"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "image/png")
				w.Write(tc.body)
			}))
			defer ts.Close()

			file, err := downloadImage(
				context.Background(),
				newLogoHTTPClient(allowAllIPs),
				ts.URL+"/logo.png",
			)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, file)
				return
			}
			require.NoError(t, err)
			require.Equal(t, int64(len(tc.body)), file.Size)
		})
	}
}
