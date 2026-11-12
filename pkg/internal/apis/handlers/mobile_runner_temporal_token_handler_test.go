// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tokenTestRunnerID = "usera-s-organization/lab-runner"

func (f registrationFixture) requestTemporalToken(
	t *testing.T,
	auth *core.Record,
	runnerID string,
) (*core.RequestEvent, error) {
	t.Helper()

	return f.requestScopedTemporalToken(t, auth, runnerID, "")
}

func (f registrationFixture) requestScopedTemporalToken(
	t *testing.T,
	auth *core.Record,
	runnerID string,
	scope string,
) (*core.RequestEvent, error) {
	t.Helper()

	event := performMobileRunnerRequest(
		t,
		f.app,
		auth,
		"/api/mobile-runner/temporal-token",
		MobileRunnerTemporalTokenRequest{RunnerID: runnerID, Scope: scope},
	)
	return event, HandleMobileRunnerTemporalToken()(event)
}

func (f registrationFixture) setRunnerField(t *testing.T, field string, value any) {
	t.Helper()

	record, err := canonify.Resolve(f.app, "/"+tokenTestRunnerID)
	require.NoError(t, err)
	record.Set(field, value)
	require.NoError(t, f.app.Save(record))
}

func orgNamespace(t *testing.T, f registrationFixture, orgID string) string {
	t.Helper()

	org, err := f.app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	return org.GetString("canonified_name")
}

// tokenClaims decodes the issued token and returns its subject and permissions.
func tokenClaims(t *testing.T, event *core.RequestEvent) (string, []string) {
	t.Helper()

	recorder := responseRecorder(t, event)
	require.Equal(t, http.StatusOK, recorder.Code)
	body := decodeJSONBody(t, recorder)
	assert.InDelta(t, 900, body["expires_in"], 0)

	token, ok := body["token"].(string)
	require.True(t, ok)
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(token, claims)
	require.NoError(t, err)

	raw, ok := claims["permissions"].([]any)
	require.True(t, ok)
	permissions := make([]string, 0, len(raw))
	for _, p := range raw {
		permissions = append(permissions, p.(string))
	}
	sub, _ := claims["sub"].(string)
	return sub, permissions
}

func TestHandleMobileRunnerTemporalToken(t *testing.T) {
	t.Run("own non-published runner gets its organization only", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		f.setRunnerField(t, "published", false)

		event, err := f.requestTemporalToken(t, f.user, tokenTestRunnerID)
		require.NoError(t, err)
		sub, permissions := tokenClaims(t, event)
		assert.Equal(t, "mobile-runner:"+tokenTestRunnerID, sub)
		assert.Equal(t, []string{"usera-s-organization:write"}, permissions)
	})

	t.Run("runner with an enabled agent also gets its agent namespace", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		f.setRunnerField(t, "published", false)
		f.setRunnerField(t, "agent_enabled", true)
		record, err := canonify.Resolve(f.app, "/"+tokenTestRunnerID)
		require.NoError(t, err)

		event, err := f.requestTemporalToken(t, f.user, tokenTestRunnerID)
		require.NoError(t, err)
		_, permissions := tokenClaims(t, event)
		assert.Equal(
			t,
			[]string{"runner-" + record.Id + ":write", "usera-s-organization:write"},
			permissions,
		)
	})

	t.Run("published runner gets published organizations but not default", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		f.setRunnerField(t, "published", true)
		setOrganizationPublished(t, f.app, f.otherOrg, true)

		event, err := f.requestTemporalToken(t, f.user, tokenTestRunnerID)
		require.NoError(t, err)
		_, permissions := tokenClaims(t, event)
		assert.Contains(t, permissions, "usera-s-organization:write")
		assert.Contains(t, permissions, orgNamespace(t, f, f.otherOrg)+":write")
		assert.NotContains(t, permissions, "default:write")
		assert.IsNonDecreasing(t, permissions)
	})

	t.Run("superuser gets default and every organization for an admin-managed runner",
		func(t *testing.T) {
			f := setupRegistrationFixture(t)
			setMobileRunnerAdminManaged(t, f.app, tokenTestRunnerID, true)

			event, err := f.requestTemporalToken(t, f.superuser, tokenTestRunnerID)
			require.NoError(t, err)
			_, permissions := tokenClaims(t, event)

			orgs, err := f.app.FindAllRecords("organizations")
			require.NoError(t, err)
			want := []string{"default:write"}
			for _, org := range orgs {
				if ns := org.GetString("canonified_name"); ns != "" {
					want = append(want, ns+":write")
				}
			}
			assert.ElementsMatch(t, want, permissions)
			assert.IsIncreasing(t, permissions)
		})

	errorCases := []struct {
		name       string
		prepare    func(t *testing.T, f registrationFixture)
		auth       func(f registrationFixture) *core.Record
		runnerID   string
		wantCode   int
		wantReason string
	}{
		{
			name: "user cannot get a token for an admin-managed runner",
			prepare: func(t *testing.T, f registrationFixture) {
				setMobileRunnerAdminManaged(t, f.app, tokenTestRunnerID, true)
			},
			auth:       func(f registrationFixture) *core.Record { return f.user },
			runnerID:   tokenTestRunnerID,
			wantCode:   http.StatusForbidden,
			wantReason: "admin_managed_runner",
		},
		{
			name:       "user cannot get a token for another organization's runner",
			auth:       func(f registrationFixture) *core.Record { return f.otherUser },
			runnerID:   tokenTestRunnerID,
			wantCode:   http.StatusForbidden,
			wantReason: "runner_owner_mismatch",
		},
		{
			name: "disabled runner gets no token",
			prepare: func(t *testing.T, f registrationFixture) {
				f.setRunnerField(t, "disabled", true)
			},
			auth:       func(f registrationFixture) *core.Record { return f.user },
			runnerID:   tokenTestRunnerID,
			wantCode:   http.StatusForbidden,
			wantReason: "runner_disabled",
		},
		{
			name:       "unknown runner",
			auth:       func(f registrationFixture) *core.Record { return f.user },
			runnerID:   "usera-s-organization/missing-runner",
			wantCode:   http.StatusNotFound,
			wantReason: "mobile_runner_not_found",
		},
		{
			name: "organization member without the owner role gets no token",
			prepare: func(t *testing.T, f registrationFixture) {
				memberRole, err := f.app.FindFirstRecordByData("orgRoles", "name", "member")
				require.NoError(t, err)
				authorization, err := f.app.FindFirstRecordByData(
					"orgAuthorizations",
					"user",
					f.user.Id,
				)
				require.NoError(t, err)
				authorization.Set("role", memberRole.Id)
				require.NoError(t, f.app.Save(authorization))
			},
			auth:       func(f registrationFixture) *core.Record { return f.user },
			runnerID:   tokenTestRunnerID,
			wantCode:   http.StatusForbidden,
			wantReason: "runner_owner_role_required",
		},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			f := setupRegistrationFixture(t)
			if tc.prepare != nil {
				tc.prepare(t, f)
			}
			_, err := f.requestTemporalToken(t, tc.auth(f), tc.runnerID)
			requireRegistrationAPIError(t, err, tc.wantCode, "runner_id", tc.wantReason)
		})
	}
}

func TestHandleMobileRunnerTemporalTokenReservedNamespace(t *testing.T) {
	f := setupRegistrationFixture(t)
	f.setRunnerField(t, "published", false)
	// A legacy organization holding a reserved namespace; the canonify update
	// hook would rewrite a normal save, so write the column directly.
	_, err := f.app.DB().
		NewQuery("UPDATE organizations SET canonified_name='default' WHERE id={:id}").
		Bind(dbx.Params{"id": f.orgID}).
		Execute()
	require.NoError(t, err)

	_, err = f.requestTemporalToken(t, f.user, "default/lab-runner")
	requireRegistrationAPIError(
		t,
		err,
		http.StatusConflict,
		"organizations",
		"organization_namespace_reserved",
	)
}

func TestHandleMobileRunnerTemporalTokenAgentScope(t *testing.T) {
	t.Run("agent scope grants only the agent namespace", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		f.setRunnerField(t, "published", true)
		f.setRunnerField(t, "agent_enabled", true)
		record, err := canonify.Resolve(f.app, "/"+tokenTestRunnerID)
		require.NoError(t, err)

		event, err := f.requestScopedTemporalToken(
			t,
			f.user,
			tokenTestRunnerID,
			MobileRunnerTemporalTokenScopeAgent,
		)
		require.NoError(t, err)
		sub, permissions := tokenClaims(t, event)
		assert.Equal(t, "mobile-runner-agent:"+tokenTestRunnerID, sub)
		assert.Equal(t, []string{"runner-" + record.Id + ":write"}, permissions)
	})

	t.Run("agent scope needs an enabled agent", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		f.setRunnerField(t, "agent_enabled", false)

		_, err := f.requestScopedTemporalToken(
			t,
			f.user,
			tokenTestRunnerID,
			MobileRunnerTemporalTokenScopeAgent,
		)
		requireRegistrationAPIError(t, err, http.StatusForbidden, "scope", "agent_not_enabled")
	})

	t.Run("agent scope keeps the runner checks", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		f.setRunnerField(t, "agent_enabled", true)
		f.setRunnerField(t, "disabled", true)

		_, err := f.requestScopedTemporalToken(
			t,
			f.user,
			tokenTestRunnerID,
			MobileRunnerTemporalTokenScopeAgent,
		)
		requireRegistrationAPIError(t, err, http.StatusForbidden, "runner_id", "runner_disabled")
	})

	t.Run("unknown scope is rejected", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		f.setRunnerField(t, "agent_enabled", true)

		_, err := f.requestScopedTemporalToken(t, f.user, tokenTestRunnerID, "runner")
		requireRegistrationAPIError(t, err, http.StatusBadRequest, "scope", "invalid_request")
	})
}
