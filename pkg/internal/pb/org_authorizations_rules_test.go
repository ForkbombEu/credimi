// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rulesTestUserC = "userC@example.org"

// TestOrgAuthorizationsReadRulesScopeToMembers checks that an orgAuthorizations
// row is readable only by its own user or by members of the same organization.
func TestOrgAuthorizationsReadRulesScopeToMembers(t *testing.T) {
	app := newRulesTestApp(t)
	mux := buildRulesTestMux(t, app)

	userAAuth := findOrgAuthorization(t, app, jsHooksUserA, jsHooksUserAOrg)
	userBAuth := findOrgAuthorization(t, app, rulesTestUserB, jsHooksUserBOrg)
	// userC joins userA's organization as a plain member.
	userCAuth := addOrgMember(t, app, rulesTestUserC, jsHooksUserAOrg)

	listOrg := func(orgID string) string {
		query := url.Values{
			"expand": {"user"},
			"filter": {"organization='" + orgID + "'"},
		}
		return "/api/collections/orgAuthorizations/records?" + query.Encode()
	}
	view := func(id string) string {
		return "/api/collections/orgAuthorizations/records/" + id
	}

	listCases := []struct {
		name    string
		email   string
		path    string
		wantIDs []string
	}{
		{
			name:    "anonymous caller lists no membership",
			path:    "/api/collections/orgAuthorizations/records",
			wantIDs: []string{},
		},
		{
			name:    "anonymous caller cannot list another organization",
			path:    listOrg(jsHooksUserBOrg),
			wantIDs: []string{},
		},
		{
			name:    "other tenant cannot list another organization",
			email:   jsHooksUserA,
			path:    listOrg(jsHooksUserBOrg),
			wantIDs: []string{},
		},
		{
			name:    "owner lists every member of their organization",
			email:   jsHooksUserA,
			path:    listOrg(jsHooksUserAOrg),
			wantIDs: []string{userAAuth.Id, userCAuth.Id},
		},
		{
			name:    "member lists every member of their organization",
			email:   rulesTestUserC,
			path:    listOrg(jsHooksUserAOrg),
			wantIDs: []string{userAAuth.Id, userCAuth.Id},
		},
		{
			name:    "unfiltered list returns only the caller's organizations",
			email:   rulesTestUserB,
			path:    "/api/collections/orgAuthorizations/records",
			wantIDs: []string{userBAuth.Id},
		},
	}
	for _, tc := range listCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveRulesRequest(mux, http.MethodGet, tc.path, "", userToken(t, app, tc.email))
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

			var page struct {
				Items []struct {
					ID string `json:"id"`
				} `json:"items"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
			gotIDs := make([]string, 0, len(page.Items))
			for _, item := range page.Items {
				gotIDs = append(gotIDs, item.ID)
			}
			assert.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}

	viewCases := []struct {
		name     string
		email    string
		id       string
		wantCode int
	}{
		{
			name:     "anonymous caller cannot view a membership",
			id:       userBAuth.Id,
			wantCode: http.StatusNotFound,
		},
		{
			name:     "other tenant cannot view a membership",
			email:    jsHooksUserA,
			id:       userBAuth.Id,
			wantCode: http.StatusNotFound,
		},
		{
			name:     "user views their own membership",
			email:    rulesTestUserB,
			id:       userBAuth.Id,
			wantCode: http.StatusOK,
		},
		{
			name:     "member views another member of the same organization",
			email:    rulesTestUserC,
			id:       userAAuth.Id,
			wantCode: http.StatusOK,
		},
	}
	for _, tc := range viewCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveRulesRequest(
				mux,
				http.MethodGet,
				view(tc.id),
				"",
				userToken(t, app, tc.email),
			)
			require.Equal(t, tc.wantCode, rec.Code, rec.Body.String())
		})
	}
}

func findOrgAuthorization(t *testing.T, app core.App, email, orgID string) *core.Record {
	t.Helper()
	user, err := app.FindAuthRecordByEmail("users", email)
	require.NoError(t, err)
	record, err := app.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user = {:user} && organization = {:org}",
		map[string]any{"user": user.Id, "org": orgID},
	)
	require.NoError(t, err)
	return record
}

func addOrgMember(t *testing.T, app *tests.TestApp, email, orgID string) *core.Record {
	t.Helper()
	user, err := app.FindAuthRecordByEmail("users", email)
	require.NoError(t, err)
	role, err := app.FindFirstRecordByFilter("orgRoles", "name = 'member'")
	require.NoError(t, err)
	coll, err := app.FindCollectionByNameOrId("orgAuthorizations")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("user", user.Id)
	record.Set("organization", orgID)
	record.Set("role", role.Id)
	require.NoError(t, app.Save(record))
	return record
}

// userToken returns an auth token for the users-collection email, or "" for an
// anonymous request.
func userToken(t *testing.T, app core.App, email string) string {
	t.Helper()
	if email == "" {
		return ""
	}
	user, err := app.FindAuthRecordByEmail("users", email)
	require.NoError(t, err)
	token, err := user.NewAuthToken()
	require.NoError(t, err)
	return token
}
