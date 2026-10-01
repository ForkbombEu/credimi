// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const (
	jsHooksDir      = "../../../pb_hooks"
	jsHooksUserA    = "userA@example.org"
	jsHooksUserAOrg = "co35481b68u3zj3"
	jsHooksUserBOrg = "3u4982xn6ah0433"
)

// setupJSHooksTestApp loads the real pb_hooks (and no JS migrations) into a
// test app, so the organization verification routes run as in production.
func setupJSHooksTestApp(t testing.TB) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	require.NoError(t, jsvm.Register(app, jsvm.Config{
		HooksDir:               jsHooksDir,
		MigrationsDir:          jsHooksDir,
		MigrationsFilesPattern: "^$",
	}))
	return app
}

func jsHooksAuthHeader(t testing.TB, app core.App, email string) map[string]string {
	t.Helper()
	user, err := app.FindAuthRecordByEmail("users", email)
	require.NoError(t, err)
	token, err := user.NewAuthToken()
	require.NoError(t, err)
	return map[string]string{"Authorization": token}
}

func TestOrganizationVerifyRoutesRejectFilterInjection(t *testing.T) {
	scenarios := []struct {
		name           string
		url            string
		body           string
		auth           bool
		expectedStatus int
		expectedBody   []string
	}{
		{
			name:           "membership: anonymous caller is rejected",
			url:            "/organizations/verify-user-membership",
			body:           `{"organizationId":"x\" || user.email~\"userA%\" || id=\""}`,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "membership: injected organizationId cannot match another row",
			url:            "/organizations/verify-user-membership",
			body:           `{"organizationId":"x\" || user.email~\"userB%\" || id=\""}`,
			auth:           true,
			expectedStatus: http.StatusOK,
			expectedBody:   []string{`"isMember":false`},
		},
		{
			name:           "membership: member of own organization",
			url:            "/organizations/verify-user-membership",
			body:           `{"organizationId":"` + jsHooksUserAOrg + `"}`,
			auth:           true,
			expectedStatus: http.StatusOK,
			expectedBody:   []string{`"isMember":true`},
		},
		{
			name:           "membership: not member of another organization",
			url:            "/organizations/verify-user-membership",
			body:           `{"organizationId":"` + jsHooksUserBOrg + `"}`,
			auth:           true,
			expectedStatus: http.StatusOK,
			expectedBody:   []string{`"isMember":false`},
		},
		{
			name:           "role: anonymous caller is rejected",
			url:            "/organizations/verify-user-role",
			body:           `{"organizationId":"x","roles":["owner\") || (user.password~\"$2a%"]}`,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "role: injected role cannot widen the match",
			url:  "/organizations/verify-user-role",
			body: `{"organizationId":"` + jsHooksUserBOrg +
				`","roles":["owner\") || (user.email~\"userB%"]}`,
			auth:           true,
			expectedStatus: http.StatusOK,
			expectedBody:   []string{`"hasRole":false`},
		},
		{
			name:           "role: non-string role is rejected",
			url:            "/organizations/verify-user-role",
			body:           `{"organizationId":"` + jsHooksUserAOrg + `","roles":[{"x":1}]}`,
			auth:           true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "role: owner of own organization",
			url:            "/organizations/verify-user-role",
			body:           `{"organizationId":"` + jsHooksUserAOrg + `","roles":["admin","owner"]}`,
			auth:           true,
			expectedStatus: http.StatusOK,
			expectedBody:   []string{`"hasRole":true`},
		},
		{
			name:           "role: owner lacks member role",
			url:            "/organizations/verify-user-role",
			body:           `{"organizationId":"` + jsHooksUserAOrg + `","roles":["member"]}`,
			auth:           true,
			expectedStatus: http.StatusOK,
			expectedBody:   []string{`"hasRole":false`},
		},
	}

	for _, sc := range scenarios {
		scenario := tests.ApiScenario{
			Name:            sc.name,
			Method:          http.MethodPost,
			URL:             sc.url,
			Body:            strings.NewReader(sc.body),
			ExpectedStatus:  sc.expectedStatus,
			ExpectedContent: sc.expectedBody,
			TestAppFactory:  setupJSHooksTestApp,
		}
		if sc.expectedBody == nil {
			scenario.ExpectedContent = []string{`"status":`}
		}
		if sc.auth {
			scenario.BeforeTestFunc = func(t testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				scenario.Headers = jsHooksAuthHeader(t, app, jsHooksUserA)
			}
		}
		scenario.Test(t)
	}
}
