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
		err := app.OnRecordCreateRequest("users").
			Trigger(event, func(e *core.RecordRequestEvent) error {
				return nil
			})

		require.Error(t, err)
		require.ErrorContains(t, err, "Captcha verification failed")
	})

	t.Run("requires captcha for password login", func(t *testing.T) {
		app := pocketbase.New()
		HookTurnstileVerification(app)

		event := newUserPasswordAuthRequestEvent(t, app)
		err := app.OnRecordAuthWithPasswordRequest("users").
			Trigger(event, func(e *core.RecordAuthWithPasswordRequestEvent) error {
				return nil
			})

		require.Error(t, err)
		require.ErrorContains(t, err, "Captcha verification failed")
	})

	t.Run(
		"allows password login with a token when verification is not configured",
		func(t *testing.T) {
			t.Setenv("TURNSTILE_SECRET_KEY", "")

			app := pocketbase.New()
			HookTurnstileVerification(app)

			called := false
			event := newUserPasswordAuthRequestEvent(t, app)
			event.Request.Header.Set("X-Turnstile-Token", "test-token")
			err := app.OnRecordAuthWithPasswordRequest("users").
				Trigger(event, func(e *core.RecordAuthWithPasswordRequestEvent) error {
					called = true
					return nil
				})

			require.NoError(t, err)
			require.True(t, called)
		},
	)

	t.Run("bypasses captcha for OAuth2-created users", func(t *testing.T) {
		app := pocketbase.New()
		HookTurnstileVerification(app)

		called := false
		event := newUserCreateRequestEvent(t, app, core.RequestInfoContextOAuth2)
		err := app.OnRecordCreateRequest("users").
			Trigger(event, func(e *core.RecordRequestEvent) error {
				called = true
				return nil
			})

		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("requires captcha for OAuth2 registration", func(t *testing.T) {
		app := pocketbase.New()
		HookTurnstileVerification(app)

		event := newOAuth2AuthRequestEvent(t, app)
		event.IsNewRecord = true
		err := app.OnRecordAuthWithOAuth2Request("users").
			Trigger(event, func(e *core.RecordAuthWithOAuth2RequestEvent) error {
				return nil
			})

		require.Error(t, err)
		require.ErrorContains(t, err, "Captcha verification failed")
	})

	t.Run("requires captcha for OAuth2 login", func(t *testing.T) {
		app := pocketbase.New()
		HookTurnstileVerification(app)

		event := newOAuth2AuthRequestEvent(t, app)
		event.IsNewRecord = false
		err := app.OnRecordAuthWithOAuth2Request("users").
			Trigger(event, func(e *core.RecordAuthWithOAuth2RequestEvent) error {
				return nil
			})

		require.Error(t, err)
		require.ErrorContains(t, err, "Captcha verification failed")
	})

	t.Run(
		"accepts OAuth2 registration with captcha when secret is not configured",
		func(t *testing.T) {
			t.Setenv("TURNSTILE_SECRET_KEY", "")

			app := pocketbase.New()
			HookTurnstileVerification(app)

			called := false
			event := newOAuth2AuthRequestEvent(t, app)
			event.IsNewRecord = true
			event.Request.Header.Set("X-Turnstile-Token", "test-token")
			err := app.OnRecordAuthWithOAuth2Request("users").
				Trigger(event, func(e *core.RecordAuthWithOAuth2RequestEvent) error {
					called = true
					return nil
				})

			require.NoError(t, err)
			require.True(t, called)
		},
	)
}

func TestIsOAuth2RecordCreateRequest(t *testing.T) {
	app := pocketbase.New()

	require.False(t, isOAuth2RecordCreateRequest(nil))
	require.False(t, isOAuth2RecordCreateRequest(&core.RecordRequestEvent{}))
	require.False(
		t,
		isOAuth2RecordCreateRequest(
			newUserCreateRequestEvent(t, app, core.RequestInfoContextDefault),
		),
	)
	require.True(
		t,
		isOAuth2RecordCreateRequest(
			newUserCreateRequestEvent(t, app, core.RequestInfoContextOAuth2),
		),
	)
}

func newUserCreateRequestEvent(
	t testing.TB,
	app core.App,
	context string,
) *core.RecordRequestEvent {
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

func newOAuth2AuthRequestEvent(
	t testing.TB,
	app core.App,
) *core.RecordAuthWithOAuth2RequestEvent {
	t.Helper()

	users := core.NewAuthCollection("users")

	req := httptest.NewRequest(http.MethodPost, "/api/collections/users/auth-with-oauth2", nil)
	rec := httptest.NewRecorder()
	requestEvent := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	event := &core.RecordAuthWithOAuth2RequestEvent{
		RequestEvent: requestEvent,
	}
	event.Collection = users

	return event
}

func newUserPasswordAuthRequestEvent(
	t testing.TB,
	app core.App,
) *core.RecordAuthWithPasswordRequestEvent {
	t.Helper()

	users := core.NewAuthCollection("users")
	req := httptest.NewRequest(http.MethodPost, "/api/collections/users/auth-with-password", nil)
	rec := httptest.NewRecorder()
	requestEvent := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	event := &core.RecordAuthWithPasswordRequestEvent{
		RequestEvent: requestEvent,
	}
	event.Collection = users

	return event
}
