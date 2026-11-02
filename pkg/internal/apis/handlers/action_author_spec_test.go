// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const (
	authoringOnboardingAction = authoringNamespace + "/authoring-handler-wallet/onboarding"
	authoringGetPIDAction     = authoringNamespace + "/authoring-handler-wallet/get-pid"
	authoringOtherWallet      = authoringNamespace + "/other-authoring-wallet/foreign-onboarding"
	authoringForeignOffer     = authoringForeignNS + "/foreign-issuer/foreign-pid"
)

// setupActionAuthorSpecApp adds to setupActionAuthorApp the handler wallet's
// actions (an owned onboarding, an owned get-pid with ${deeplink}, a
// published and an unpublished action of userB's organization), an action of
// another wallet of userA's organization, and an unpublished credential of
// userB's organization.
func setupActionAuthorSpecApp(t testing.TB) *tests.TestApp {
	app := setupActionAuthorApp(t)
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	foreignOrg, err := app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	wallet, err := app.FindFirstRecordByData("wallets", "name", "Authoring handler wallet")
	require.NoError(t, err)
	save := func(collection string, fields map[string]any) *core.Record {
		c, err := app.FindCollectionByNameOrId(collection)
		require.NoError(t, err)
		record := core.NewRecord(c)
		for key, value := range fields {
			record.Set(key, value)
		}
		require.NoError(t, app.Save(record))
		return record
	}
	action := func(walletID, owner, name, category, code string, published bool) {
		save("wallet_actions", map[string]any{
			"owner":     owner,
			"wallet":    walletID,
			"name":      name,
			"category":  category,
			"code":      code,
			"published": published,
		})
	}
	action(
		wallet.Id,
		org.Id,
		"Onboarding",
		"onboarding",
		"appId: eu.wallet\n---\n- launchApp\n",
		false,
	)
	action(wallet.Id, org.Id, "Get PID", "get-credential-generic",
		"appId: eu.wallet\n---\n- openLink: ${deeplink}\n", false)
	action(
		wallet.Id,
		foreignOrg.Id,
		"Foreign published",
		"other",
		"appId: eu.wallet\n---\n- back\n",
		true,
	)
	action(
		wallet.Id,
		foreignOrg.Id,
		"Foreign private",
		"other",
		"appId: eu.wallet\n---\n- back\n",
		false,
	)
	otherWallet := save(
		"wallets",
		map[string]any{"owner": org.Id, "name": "Other authoring wallet"},
	)
	action(
		otherWallet.Id,
		org.Id,
		"Foreign onboarding",
		"onboarding",
		"appId: x\n---\n- launchApp\n",
		true,
	)
	issuer := save("credential_issuers", map[string]any{
		"owner": foreignOrg.Id,
		"name":  "Foreign issuer",
		"url":   "https://issuer.example.test",
	})
	save("credentials", map[string]any{
		"owner":             foreignOrg.Id,
		"credential_issuer": issuer.Id,
		"name":              "Foreign PID",
	})
	return app
}

func specSessionBody(deviceID, version, spec string) string {
	return `{"device_id":"` + deviceID + `","wallet_version_identifier":"` + version + `",` + spec + `}`
}

func TestActionAuthorStartChecksSpec(t *testing.T) {
	token := authoringToken(t, "userA")
	android := authoringAdminRunner + "/android-emulator"
	for _, tc := range []struct {
		name     string
		device   string
		version  string
		spec     string
		expected string
	}{
		{
			name:     "prerequisite of another wallet",
			spec:     `"slot":"get-credential","prerequisites":[{"action":"` + authoringOtherWallet + `"}]`,
			expected: "prerequisite " + authoringOtherWallet + " belongs to another wallet",
		},
		{
			name: "unpublished credential of another organization",
			spec: `"slot":"get-credential","prerequisites":[{"action":"` + authoringOnboardingAction +
				`"}],"parameters":[{"name":"deeplink","source":{"kind":"credential_offer","identifier":"` +
				authoringForeignOffer + `"}}]`,
			expected: authoringForeignOffer + " not found",
		},
		{
			name:     "unpublished prerequisite of another organization",
			spec:     `"slot":"other","prerequisites":[{"action":"` + authoringNamespace + `/authoring-handler-wallet/foreign-private"}]`,
			expected: authoringNamespace + "/authoring-handler-wallet/foreign-private not found",
		},
		{
			name:     "prerequisite parameters",
			spec:     `"slot":"verify-credential","prerequisites":[{"action":"` + authoringGetPIDAction + `"}]`,
			expected: "prerequisite " + authoringGetPIDAction + " needs parameters [deeplink], got []",
		},
		{
			name:     "browser on iOS",
			device:   authoringAdminRunner + "/ios-simulator",
			version:  authoringBothVersion,
			spec:     `"slot":"other","browser":true`,
			expected: "browser flows need an Android device",
		},
		{
			name:     "partial spec without a slot",
			spec:     `"outcome":"rejection"`,
			expected: "slot: must be one of",
		},
		{
			name:     "rejection on onboarding",
			spec:     `"slot":"onboarding","outcome":"rejection"`,
			expected: "outcome: the onboarding slot has no rejection outcome",
		},
	} {
		enqueued := stubActionAuthorQueue(t, true)
		device, version := android, authoringAndroidVersion
		if tc.device != "" {
			device, version = tc.device, tc.version
		}
		(&tests.ApiScenario{
			Name:            tc.name,
			Method:          http.MethodPost,
			URL:             "/api/action-author/sessions",
			Body:            strings.NewReader(specSessionBody(device, version, tc.spec)),
			Headers:         map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"invalid_request"`, tc.expected},
			TestAppFactory:  setupActionAuthorSpecApp,
		}).Test(t)
		require.Empty(t, enqueued.TicketID, tc.name)
	}
}

func TestActionAuthorStartQueuesSpec(t *testing.T) {
	token := authoringToken(t, "userA")
	enqueued := stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:   "verify credential with prerequisites",
		Method: http.MethodPost,
		URL:    "/api/action-author/sessions",
		Body: strings.NewReader(specSessionBody(
			authoringAdminRunner+"/android-emulator",
			authoringAndroidVersion,
			`"slot":"verify-credential","browser":true,"prerequisites":[{"action":"/`+
				authoringOnboardingAction+`"},{"action":"`+authoringGetPIDAction+
				`","parameters":[{"name":"deeplink","source":{"kind":"literal","value":"openid://x"}}]}]`,
		)),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"ticket_id"`},
		TestAppFactory:  setupActionAuthorSpecApp,
	}).Test(t)
	require.NotEmpty(t, enqueued.TicketID)
	require.Equal(t, "verify-credential", enqueued.ActionAuthor["slot"])
	require.Equal(t, "success", enqueued.ActionAuthor["outcome"])
	require.Equal(t, true, enqueued.ActionAuthor["browser"])
	require.Equal(t, []any{
		map[string]any{"action": authoringOnboardingAction},
		map[string]any{
			"action": authoringGetPIDAction,
			"parameters": []any{map[string]any{
				"name":   "deeplink",
				"source": map[string]any{"kind": "literal", "value": "openid://x"},
			}},
		},
	}, enqueued.ActionAuthor["prerequisites"])
}

func TestActionAuthorStartQueuesEmptySpec(t *testing.T) {
	token := authoringToken(t, "userA")
	enqueued := stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:   "plan proposed in the chat",
		Method: http.MethodPost,
		URL:    "/api/action-author/sessions",
		Body: strings.NewReader(`{"device_id":"` + authoringAdminRunner +
			`/android-emulator","wallet_version_identifier":"` + authoringAndroidVersion + `"}`),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"ticket_id"`},
		TestAppFactory:  setupActionAuthorSpecApp,
	}).Test(t)
	require.NotEmpty(t, enqueued.TicketID)
	require.Equal(t, "", enqueued.ActionAuthor["slot"])
	for _, field := range []string{"outcome", "browser", "prerequisites", "parameters"} {
		require.NotContains(t, enqueued.ActionAuthor, field)
	}
}

func TestActionAuthorWalletActionsListsOwnedAndPublished(t *testing.T) {
	token := authoringToken(t, "userA")
	var response ActionAuthorWalletActionsResponse
	(&tests.ApiScenario{
		Name:   "owned and published actions of the wallet",
		Method: http.MethodGet,
		URL: "/api/action-author/wallet-actions?wallet_version_identifier=" +
			authoringAndroidVersion + "&platform=android",
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"actions"`},
		TestAppFactory:  setupActionAuthorSpecApp,
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(body, &response))
		},
	}).Test(t)
	require.Equal(t, []actionauthor.WalletAction{
		{
			Path:       authoringGetPIDAction,
			Name:       "Get PID",
			Category:   "get-credential-generic",
			Parameters: []string{"deeplink"},
		},
		{
			Path:       authoringOnboardingAction,
			Name:       "Onboarding",
			Category:   "onboarding",
			Parameters: []string{},
		},
		{
			Path:       authoringNamespace + "/authoring-handler-wallet/foreign-published",
			Name:       "Foreign published",
			Category:   "other",
			Published:  true,
			Parameters: []string{},
		},
	}, response.Actions)

	for _, tc := range []struct {
		name   string
		user   string
		query  string
		status int
	}{
		{name: "missing platform", user: "userA", query: "wallet_version_identifier=" + authoringAndroidVersion, status: http.StatusBadRequest},
		{name: "unknown platform", user: "userA", query: "wallet_version_identifier=" + authoringAndroidVersion + "&platform=web", status: http.StatusBadRequest},
		{name: "platform without installer", user: "userA", query: "wallet_version_identifier=" + authoringAndroidVersion + "&platform=ios", status: http.StatusBadRequest},
		{name: "another organization", user: "userB", query: "wallet_version_identifier=" + authoringAndroidVersion + "&platform=android", status: http.StatusForbidden},
	} {
		(&tests.ApiScenario{
			Name:            tc.name,
			Method:          http.MethodGet,
			URL:             "/api/action-author/wallet-actions?" + tc.query,
			Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, tc.user)},
			ExpectedStatus:  tc.status,
			ExpectedContent: []string{`"action_author"`},
			TestAppFactory:  setupActionAuthorSpecApp,
		}).Test(t)
	}
}
