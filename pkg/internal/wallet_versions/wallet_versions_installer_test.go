// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package walletversions

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const internalAdminAPIKey = "internal-test-api-key"

func TestNonDownloadableInstallersAreOwnerOnly(t *testing.T) {
	orgID, err := getTestOrgID()
	require.NoError(t, err)
	app := setupWalletVersionsApp(orgID)(t)
	defer app.Cleanup()
	seedInternalAdminKey(t, app)
	mux := buildTestMux(t, app)

	version, err := app.FindRecordById("wallet_versions", "record123456789")
	require.NoError(t, err)
	filePath := "/api/files/wallet_versions/" + version.Id + "/" +
		version.GetString("android_installer")

	ownerFileToken := fileToken(t, app, "userA@example.org")
	otherFileToken := fileToken(t, app, "userB@example.org")
	ownerAuthToken, err := getUserToken("userA@example.org")
	require.NoError(t, err)

	t.Run("anonymous cannot filter on installers", func(t *testing.T) {
		query := url.Values{"filter": {"android_installer~'test_app_%'"}}
		rec := serve(mux, get("/api/collections/wallet_versions/records?"+query.Encode()))
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})

	denied := map[string]*http.Request{
		"anonymous":                     get(filePath),
		"other organization file token": get(filePath + "?token=" + otherFileToken),
		"invalid api key": withHeader(
			get(filePath), "Credimi-Api-Key", "wrong-key",
		),
	}
	for name, req := range denied {
		t.Run(name+" cannot download", func(t *testing.T) {
			rec := serve(mux, req)
			require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		})
	}

	allowed := map[string]*http.Request{
		"owner file token": get(filePath + "?token=" + ownerFileToken),
		"owner auth token": withHeader(
			get(filePath), "Authorization", ownerAuthToken,
		),
		"internal admin api key": withHeader(
			get(filePath), "Credimi-Api-Key", internalAdminAPIKey,
		),
	}
	for name, req := range allowed {
		t.Run(name+" downloads", func(t *testing.T) {
			rec := serve(mux, req)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			assert.Equal(t, "fake apk content for testing", rec.Body.String())
		})
	}

	t.Run("anonymous downloads a downloadable installer", func(t *testing.T) {
		version.Set("downloadable", true)
		require.NoError(t, app.Save(version))
		t.Cleanup(func() {
			version.Set("downloadable", false)
			require.NoError(t, app.Save(version))
		})

		rec := serve(mux, get(filePath))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	})
}

func TestOwnerCreatesWalletVersionWithInstaller(t *testing.T) {
	orgID, err := getTestOrgID()
	require.NoError(t, err)
	app := setupWalletVersionsApp(orgID)(t)
	defer app.Cleanup()
	mux := buildTestMux(t, app)

	ownerAuthToken, err := getUserToken("userA@example.org")
	require.NoError(t, err)
	existing, err := app.FindRecordById("wallet_versions", "record123456789")
	require.NoError(t, err)
	// canonified_tag is unique per wallet and the canonify hooks are not bound here.
	fields := map[string]string{
		"owner":          orgID,
		"wallet":         existing.GetString("wallet"),
		"tag":            "v2.0.0",
		"canonified_tag": "v2-0-0",
	}

	t.Run("with installer", func(t *testing.T) {
		req := multipartRequest(t, fields, "android_installer", "new.apk", "new apk")
		rec := serve(mux, withHeader(req, "Authorization", ownerAuthToken))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		created, err := app.FindFirstRecordByData("wallet_versions", "tag", "v2.0.0")
		require.NoError(t, err)
		assert.NotEmpty(t, created.GetString("android_installer"))
		assert.Contains(t, rec.Body.String(), `"android_installer":"new_`)
	})

	t.Run("without installer", func(t *testing.T) {
		fields["tag"], fields["canonified_tag"] = "v3.0.0", "v3-0-0"
		req := multipartRequest(t, fields, "", "", "")
		rec := serve(mux, withHeader(req, "Authorization", ownerAuthToken))
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})
}

func seedInternalAdminKey(t testing.TB, app *tests.TestApp) {
	t.Helper()

	superuser, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, "admin@example.org")
	require.NoError(t, err)
	coll, err := app.FindCollectionByNameOrId("api_keys")
	require.NoError(t, err)
	hash, err := bcrypt.GenerateFromPassword([]byte(internalAdminAPIKey), bcrypt.MinCost)
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("name", "internal-test-key")
	record.Set("key", string(hash))
	record.Set("superuser", superuser.Id)
	record.Set("key_type", "internal_admin")
	require.NoError(t, app.Save(record))
}

func fileToken(t testing.TB, app *tests.TestApp, email string) string {
	t.Helper()

	user, err := app.FindAuthRecordByEmail("users", email)
	require.NoError(t, err)
	token, err := user.NewFileToken()
	require.NoError(t, err)
	return token
}

func buildTestMux(t testing.TB, app *tests.TestApp) http.Handler {
	t.Helper()

	router, err := apis.NewRouter(app)
	require.NoError(t, err)

	var mux http.Handler
	serveEvent := &core.ServeEvent{App: app, Router: router}
	require.NoError(t, app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		var err error
		mux, err = e.Router.BuildMux()
		return err
	}))
	return mux
}

func get(path string) *http.Request {
	return httptest.NewRequest(http.MethodGet, path, nil)
}

func withHeader(req *http.Request, key, value string) *http.Request {
	req.Header.Set(key, value)
	return req
}

func multipartRequest(
	t testing.TB,
	fields map[string]string,
	fileField, fileName, fileContent string,
) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		require.NoError(t, writer.WriteField(key, value))
	}
	if fileField != "" {
		part, err := writer.CreateFormFile(fileField, fileName)
		require.NoError(t, err)
		_, err = part.Write([]byte(fileContent))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/collections/wallet_versions/records",
		&body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func serve(mux http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
