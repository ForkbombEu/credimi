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
)

// setupActionAuthorSpecApp adds to setupActionAuthorApp the handler wallet's
// actions (an owned onboarding, an owned get-pid with ${deeplink}, a
// published and an unpublished action of userB's organization) and an action
// of another wallet of userA's organization.
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
	return app
}

// A start body of an older client with spec fields queues a session without
// them: the agent sets each plan in the chat.
func TestActionAuthorStartIgnoresSpecFields(t *testing.T) {
	token := authoringToken(t, "userA")
	enqueued := stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:   "spec fields are ignored",
		Method: http.MethodPost,
		URL:    "/api/action-author/sessions",
		Body: strings.NewReader(`{"device_id":"` + authoringAdminRunner +
			`/android-emulator","wallet_version_identifier":"` + authoringAndroidVersion +
			`","slot":"onboarding","outcome":"rejection"}`),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"ticket_id"`},
		TestAppFactory:  setupActionAuthorSpecApp,
	}).Test(t)
	require.NotEmpty(t, enqueued.TicketID)
	for _, field := range []string{"slot", "outcome", "wait_seconds", "prerequisites", "parameters"} {
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
