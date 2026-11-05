// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func TestRequireCachedAPIKey(t *testing.T) {
	app, err := tests.NewTestApp(middlewareTestDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	require.NoError(t, err)
	superuser := core.NewRecord(superusers)
	superuser.SetEmail("cache-admin@example.org")
	superuser.SetPassword("cache-admin-password")
	require.NoError(t, app.Save(superuser))

	userKey := createAPIKeyRecord(
		t,
		app,
		apiKeyRecordInput{Plaintext: "cached-user-key", UserID: user.Id, Scope: apiKeyScopeUser},
	)
	createAPIKeyRecord(
		t,
		app,
		apiKeyRecordInput{
			Plaintext:   "cached-admin-key",
			SuperuserID: superuser.Id,
			Scope:       apiKeyScopeInternalAdmin,
		},
	)

	handler := RequireCachedAPIKey()
	call := func(key string) (int, *core.Record) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		if key != "" {
			req.Header.Set(apiKeyHeaderName, key)
		}
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}
		if err := handler.Func(e); err != nil {
			writeMiddlewareErrorResponse(t, e, err)
			return rec.Code, nil
		}
		return http.StatusOK, e.Auth
	}

	status, owner := call("cached-user-key")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, user.Id, owner.Id)
	status, owner = call("cached-user-key") // served from the cache
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, user.Id, owner.Id)

	status, owner = call("cached-admin-key")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, superuser.Id, owner.Id)
	require.Equal(t, core.CollectionNameSuperusers, owner.Collection().Name)

	// Revocation applies to a cached key on its next request.
	userKey.Set("revoked", true)
	require.NoError(t, app.Save(userKey))
	status, _ = call("cached-user-key")
	require.Equal(t, http.StatusUnauthorized, status)

	status, _ = call("not-a-key")
	require.Equal(t, http.StatusUnauthorized, status)
	status, _ = call("")
	require.Equal(t, http.StatusUnauthorized, status)
}
