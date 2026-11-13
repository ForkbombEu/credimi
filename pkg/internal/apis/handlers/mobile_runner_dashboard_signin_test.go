// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dashboardTestReturnTo = "http://192.168.1.20:8051/auth/credimi/callback"

// setDashboardClock fixes the clock of dashboard codes at now for the test.
func setDashboardClock(t *testing.T, now time.Time) {
	t.Helper()
	previous := dashboardCodeNow
	dashboardCodeNow = func() time.Time { return now }
	t.Cleanup(func() { dashboardCodeNow = previous })
}

// setupDashboardFixture is setupRegistrationFixture with an empty store of
// dashboard codes, so codes of other tests do not count against the bounds.
func setupDashboardFixture(t *testing.T) registrationFixture {
	t.Helper()
	previous := dashboardCodes
	dashboardCodes = mobilerunner.NewDashboardCodeStore()
	t.Cleanup(func() { dashboardCodes = previous })
	return setupRegistrationFixture(t)
}

// requestDashboardCode asks for a code as auth, signed in with a token when
// browser is true and with an API key otherwise.
func (f registrationFixture) requestDashboardCode(
	t *testing.T,
	auth *core.Record,
	browser bool,
	input MobileRunnerDashboardCodeRequest,
) (*core.RequestEvent, error) {
	t.Helper()

	event := performMobileRunnerRequest(
		t,
		f.app,
		auth,
		"/api/mobile-runner/dashboard-code",
		input,
	)
	if browser {
		event.Request.Header.Set("Authorization", "Bearer token")
	} else {
		event.Request.Header.Set(APIKeyHeaderName, "key")
	}
	return event, HandleMobileRunnerDashboardCode()(event)
}

func (f registrationFixture) redeemDashboardCode(
	t *testing.T,
	auth *core.Record,
	runnerID, code string,
) (*core.RequestEvent, error) {
	t.Helper()
	return f.redeemDashboardCodeAt(t, auth, runnerID, code, dashboardTestReturnTo)
}

// redeemDashboardCodeAt redeems code as the dashboard callback returnTo.
func (f registrationFixture) redeemDashboardCodeAt(
	t *testing.T,
	auth *core.Record,
	runnerID, code, returnTo string,
) (*core.RequestEvent, error) {
	t.Helper()

	event := performMobileRunnerRequest(
		t,
		f.app,
		auth,
		"/api/mobile-runner/dashboard-code/redeem",
		MobileRunnerDashboardRedeemRequest{RunnerID: runnerID, Code: code, ReturnTo: returnTo},
	)
	return event, HandleMobileRunnerDashboardRedeem()(event)
}

// mustDashboardCode issues a code for the fixture's runner to auth and returns
// it after checking the redirect.
func (f registrationFixture) mustDashboardCode(t *testing.T, auth *core.Record) string {
	t.Helper()
	return f.mustDashboardCodeFor(t, auth, dashboardTestReturnTo)
}

// mustDashboardCodeFor issues a code for the fixture's runner to auth, sent to
// returnTo, and returns it after checking the redirect.
func (f registrationFixture) mustDashboardCodeFor(
	t *testing.T,
	auth *core.Record,
	returnTo string,
) string {
	t.Helper()

	event, err := f.requestDashboardCode(t, auth, true, MobileRunnerDashboardCodeRequest{
		RunnerID: tokenTestRunnerID,
		ReturnTo: returnTo,
		State:    "st4te",
	})
	require.NoError(t, err)
	recorder := responseRecorder(t, event)
	require.Equal(t, http.StatusOK, recorder.Code)
	body := decodeJSONBody(t, recorder)
	require.Len(t, body, 1, "the answer carries only the redirect")

	redirect, err := url.Parse(body["redirect"].(string))
	require.NoError(t, err)
	want, err := url.Parse(returnTo)
	require.NoError(t, err)
	assert.Equal(t, want.Host, redirect.Host)
	assert.Equal(t, DashboardCallbackPath, redirect.Path)
	assert.Equal(t, "st4te", redirect.Query().Get("state"))
	code := redirect.Query().Get("code")
	require.Len(t, code, 43)
	return code
}

// dashboardCodeRefusal is a code request the handler refuses.
type dashboardCodeRefusal struct {
	name       string
	prepare    func(t *testing.T, f registrationFixture)
	auth       func(f registrationFixture) *core.Record
	apiKey     bool
	returnTo   string
	state      string
	wantCode   int
	wantDomain string
	wantReason string
}

// returnToRefusal is a code request whose return_to the handler refuses.
func returnToRefusal(returnTo string) dashboardCodeRefusal {
	return dashboardCodeRefusal{
		name:       "return_to " + returnTo,
		returnTo:   returnTo,
		wantCode:   http.StatusBadRequest,
		wantDomain: "return_to",
		wantReason: "invalid_return_to",
	}
}

func TestHandleMobileRunnerDashboardCode(t *testing.T) {
	refusals := []dashboardCodeRefusal{
		{
			name:       "member",
			prepare:    func(t *testing.T, f registrationFixture) { demoteToMember(t, f.app, f.user.Email()) },
			wantCode:   http.StatusForbidden,
			wantDomain: "runner_id",
			wantReason: "runner_owner_role_required",
		},
		{
			name:       "user of another organization",
			auth:       func(f registrationFixture) *core.Record { return f.otherUser },
			wantCode:   http.StatusForbidden,
			wantDomain: "runner_id",
			wantReason: "runner_owner_mismatch",
		},
		{
			name: "owner of an admin-managed runner",
			prepare: func(t *testing.T, f registrationFixture) {
				setMobileRunnerAdminManaged(t, f.app, tokenTestRunnerID, true)
			},
			wantCode:   http.StatusForbidden,
			wantDomain: "runner_id",
			wantReason: "admin_managed_runner",
		},
		{
			name:       "disabled runner",
			prepare:    func(t *testing.T, f registrationFixture) { f.setRunnerField(t, "disabled", true) },
			wantCode:   http.StatusForbidden,
			wantDomain: "runner_id",
			wantReason: "runner_disabled",
		},
		{
			name:       "API key caller",
			apiKey:     true,
			wantCode:   http.StatusForbidden,
			wantDomain: "auth",
			wantReason: "browser_session_required",
		},
		{
			name:       "state too long",
			state:      strings.Repeat("s", maxDashboardStateLength+1),
			wantCode:   http.StatusBadRequest,
			wantDomain: "mobile_runner",
			wantReason: "invalid_request",
		},
		returnToRefusal("ftp://host/auth/credimi/callback"),
		returnToRefusal("http://user:pass@host/auth/credimi/callback"),
		returnToRefusal("http://host/auth/credimi/callback#x"),
		returnToRefusal("http://host/auth/credimi/callback?next=/"),
		returnToRefusal("http://host/admin"),
		returnToRefusal("/auth/credimi/callback"),
		returnToRefusal("http:/auth/credimi/callback"),
		returnToRefusal("http://host/auth%2Fcredimi%2Fcallback"),
		returnToRefusal("http://host/auth/credimi/%63allback"),
		returnToRefusal("http://host/auth/credimi/callback/"),
	}

	for _, tc := range refusals {
		t.Run(tc.name, func(t *testing.T) {
			f := setupDashboardFixture(t)
			if tc.prepare != nil {
				tc.prepare(t, f)
			}
			auth := f.user
			if tc.auth != nil {
				auth = tc.auth(f)
			}
			input := MobileRunnerDashboardCodeRequest{
				RunnerID: tokenTestRunnerID,
				ReturnTo: dashboardTestReturnTo,
				State:    "st4te",
			}
			if tc.returnTo != "" {
				input.ReturnTo = tc.returnTo
			}
			if tc.state != "" {
				input.State = tc.state
			}

			_, err := f.requestDashboardCode(t, auth, !tc.apiKey, input)
			requireRegistrationAPIError(t, err, tc.wantCode, tc.wantDomain, tc.wantReason)
		})
	}

	t.Run("superuser signs in to an admin-managed runner", func(t *testing.T) {
		f := setupDashboardFixture(t)
		setMobileRunnerAdminManaged(t, f.app, tokenTestRunnerID, true)

		code := f.mustDashboardCode(t, f.superuser)
		event, err := f.redeemDashboardCode(t, f.superuser, tokenTestRunnerID, code)
		require.NoError(t, err)
		body := decodeJSONBody(t, responseRecorder(t, event))
		assert.Equal(t, f.superuser.Id, body["user_id"])
		assert.Equal(t, f.superuser.Email(), body["email"])
	})

	t.Run("a user with five pending codes is refused", func(t *testing.T) {
		f := setupDashboardFixture(t)
		for range 5 {
			f.mustDashboardCode(t, f.user)
		}

		_, err := f.requestDashboardCode(t, f.user, true, MobileRunnerDashboardCodeRequest{
			RunnerID: tokenTestRunnerID,
			ReturnTo: dashboardTestReturnTo,
			State:    "st4te",
		})
		requireRegistrationAPIError(
			t,
			err,
			http.StatusTooManyRequests,
			"dashboard_code",
			"too_many_codes",
		)
	})
}

func TestHandleMobileRunnerDashboardRedeem(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	t.Run("the runner redeems a code once", func(t *testing.T) {
		f := setupDashboardFixture(t)
		setDashboardClock(t, now)
		code := f.mustDashboardCode(t, f.user)

		event, err := f.redeemDashboardCode(t, f.user, tokenTestRunnerID, code)
		require.NoError(t, err)
		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, map[string]any{
			"user_id":         f.user.Id,
			"auth_collection": "users",
			"email":           f.user.Email(),
			"name":            f.user.GetString("name"),
		}, decodeJSONBody(t, recorder))

		_, err = f.redeemDashboardCode(t, f.user, tokenTestRunnerID, code)
		requireRegistrationAPIError(t, err, http.StatusBadRequest, "code", "invalid_code")
	})

	t.Run("an expired code is refused", func(t *testing.T) {
		f := setupDashboardFixture(t)
		setDashboardClock(t, now)
		code := f.mustDashboardCode(t, f.user)

		setDashboardClock(t, now.Add(61*time.Second))
		_, err := f.redeemDashboardCode(t, f.user, tokenTestRunnerID, code)
		requireRegistrationAPIError(t, err, http.StatusBadRequest, "code", "invalid_code")
	})

	t.Run("a code sent to another dashboard is refused", func(t *testing.T) {
		f := setupDashboardFixture(t)
		code := f.mustDashboardCodeFor(t, f.user, "https://evil.example/auth/credimi/callback")

		_, err := f.redeemDashboardCodeAt(t, f.user, tokenTestRunnerID, code, dashboardTestReturnTo)
		requireRegistrationAPIError(t, err, http.StatusBadRequest, "code", "invalid_code")
	})

	// The route's validation answers 400 invalid_request before the handler;
	// the handler itself never matches an empty return_to.
	t.Run("a redeem with an empty return_to is refused", func(t *testing.T) {
		f := setupDashboardFixture(t)
		code := f.mustDashboardCode(t, f.user)

		_, err := f.redeemDashboardCodeAt(t, f.user, tokenTestRunnerID, code, "")
		requireRegistrationAPIError(t, err, http.StatusBadRequest, "code", "invalid_code")
	})

	t.Run("a code of another runner is refused", func(t *testing.T) {
		f := setupDashboardFixture(t)
		code := f.mustDashboardCode(t, f.user)

		otherRunner := orgNamespace(t, f, f.otherOrg) + "/other-runner"
		_, err := f.redeemDashboardCode(t, f.otherUser, otherRunner, code)
		requireRegistrationAPIError(t, err, http.StatusBadRequest, "code", "invalid_code")
	})

	t.Run("a key without rights on the runner is refused", func(t *testing.T) {
		f := setupDashboardFixture(t)
		code := f.mustDashboardCode(t, f.user)

		_, err := f.redeemDashboardCode(t, f.otherUser, tokenTestRunnerID, code)
		requireRegistrationAPIError(
			t,
			err,
			http.StatusForbidden,
			"runner_id",
			"runner_owner_mismatch",
		)
	})

	t.Run("a user demoted after the code was issued is refused", func(t *testing.T) {
		f := setupDashboardFixture(t)
		code := f.mustDashboardCode(t, f.user)
		demoteToMember(t, f.app, f.user.Email())

		_, err := f.redeemDashboardCode(t, f.superuser, tokenTestRunnerID, code)
		apiErr := requireRegistrationAPIError(
			t,
			err,
			http.StatusForbidden,
			"runner_id",
			"runner_owner_role_required",
		)
		assert.Equal(
			t,
			"only owners of the runner's organization can sign in to its dashboard",
			apiErr.Message,
		)
	})

	t.Run("a member's key is refused naming the dashboard", func(t *testing.T) {
		f := setupDashboardFixture(t)
		code := f.mustDashboardCode(t, f.user)
		demoteToMember(t, f.app, f.user.Email())

		_, err := f.redeemDashboardCode(t, f.user, tokenTestRunnerID, code)
		apiErr := requireRegistrationAPIError(
			t,
			err,
			http.StatusForbidden,
			"runner_id",
			"runner_owner_role_required",
		)
		assert.Contains(t, apiErr.Message, "dashboard")
		assert.NotContains(t, apiErr.Message, "Temporal")
	})

	t.Run(
		"an owner's key of an admin-managed runner is refused naming the dashboard",
		func(t *testing.T) {
			f := setupDashboardFixture(t)
			setMobileRunnerAdminManaged(t, f.app, tokenTestRunnerID, true)
			code := f.mustDashboardCode(t, f.superuser)

			_, err := f.redeemDashboardCode(t, f.user, tokenTestRunnerID, code)
			apiErr := requireRegistrationAPIError(
				t,
				err,
				http.StatusForbidden,
				"runner_id",
				"admin_managed_runner",
			)
			assert.Contains(t, apiErr.Message, "dashboard")
			assert.NotContains(t, apiErr.Message, "Temporal")
		},
	)
}
