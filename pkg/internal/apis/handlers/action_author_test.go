// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

// setupActionAuthorApp seeds an Android-only build owned by userA's organization.
func setupActionAuthorApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	canonify.RegisterCanonifyHooks(app)
	ActionAuthorRoutes.Add(app)
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", org.Id)
	wallet.Set("name", "Authoring handler wallet")
	require.NoError(t, app.Save(wallet))
	versions, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	version := core.NewRecord(versions)
	version.Set("owner", org.Id)
	version.Set("wallet", wallet.Id)
	version.Set("tag", "android-only")
	apk, err := filesystem.NewFileFromBytes([]byte("apk"), "selected.apk")
	require.NoError(t, err)
	version.Set("android_installer", []*filesystem.File{apk})
	require.NoError(t, app.Save(version))
	return app
}

func TestActionAuthorStartRequiresPlatform(t *testing.T) {
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)
	identifier := "usera-s-organization/authoring-handler-wallet/android-only"
	scenarios := []struct {
		name     string
		body     string
		expected []string
	}{
		{
			name:     "missing platform",
			body:     `{"wallet_version_identifier":"` + identifier + `","slot":"onboarding"}`,
			expected: []string{"ActionAuthorStartRequest.Platform", "'required' tag"},
		},
		{
			name:     "unsupported platform",
			body:     `{"wallet_version_identifier":"` + identifier + `","platform":"windows","slot":"onboarding"}`,
			expected: []string{"ActionAuthorStartRequest.Platform", "'oneof' tag"},
		},
		{
			name:     "platform without installer",
			body:     `{"wallet_version_identifier":"` + identifier + `","platform":"ios","slot":"onboarding"}`,
			expected: []string{"selected version has no iOS app"},
		},
	}
	for _, scenario := range scenarios {
		(&tests.ApiScenario{
			Name:            scenario.name,
			Method:          http.MethodPost,
			URL:             "/api/action-author/sessions",
			Body:            strings.NewReader(scenario.body),
			Headers:         map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: scenario.expected,
			TestAppFactory:  setupActionAuthorApp,
		}).Test(t)
	}
}
