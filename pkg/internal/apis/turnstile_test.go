// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func TestHookTurnstileVerification(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "secret")

	t.Run("requires captcha for direct user creation", func(t *testing.T) {
		app := pocketbase.New()
		HookTurnstileVerification(app)

		event := newUserCreateRequestEvent(t, app, core.RequestInfoContextDefault)
		err := app.OnRecordCreateRequest("users").Trigger(event, func(e *core.RecordRequestEvent) error {
			return nil
		})

		require.Error(t, err)
		require.ErrorContains(t, err, "Captcha verification failed")
	})

	t.Run("bypasses captcha for OAuth2-created users", func(t *testing.T) {
		app := pocketbase.New()
		HookTurnstileVerification(app)

		called := false
		event := newUserCreateRequestEvent(t, app, core.RequestInfoContextOAuth2)
		err := app.OnRecordCreateRequest("users").Trigger(event, func(e *core.RecordRequestEvent) error {
			called = true
			return nil
		})

		require.NoError(t, err)
		require.True(t, called)
	})
}

func TestIsOAuth2RecordCreateRequest(t *testing.T) {
	app := pocketbase.New()

	require.False(t, isOAuth2RecordCreateRequest(nil))
	require.False(t, isOAuth2RecordCreateRequest(&core.RecordRequestEvent{}))
	require.False(t, isOAuth2RecordCreateRequest(newUserCreateRequestEvent(t, app, core.RequestInfoContextDefault)))
	require.True(t, isOAuth2RecordCreateRequest(newUserCreateRequestEvent(t, app, core.RequestInfoContextOAuth2)))
}

func newUserCreateRequestEvent(t testing.TB, app core.App, context string) *core.RecordRequestEvent {
	t.Helper()

	users := core.NewAuthCollection("users")

	req := httptest.NewRequest(http.MethodPost, "/api/collections/users/records", nil)
	rec := httptest.NewRecorder()
	requestEvent := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}
	requestEvent.Set(core.RequestEventKeyInfoContext, context)

	event := &core.RecordRequestEvent{
		RequestEvent: requestEvent,
		Record:       core.NewRecord(users),
	}
	event.Collection = users

	return event
}
