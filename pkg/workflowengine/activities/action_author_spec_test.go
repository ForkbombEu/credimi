// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

// seedAuthorAction adds an action to the wallet of the session's build and
// returns its canonified path.
func seedAuthorAction(
	t *testing.T,
	app core.App,
	session actionauthor.BuildRequest,
	name, code string,
) string {
	t.Helper()
	selection, err := actionauthor.Select(app, "", session)
	require.NoError(t, err)
	collection, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("owner", selection.Organization.Id)
	record.Set("wallet", selection.Wallet.Id)
	record.Set("name", name)
	record.Set("code", code)
	record.Set("category", "onboarding")
	require.NoError(t, app.Save(record))
	return selection.Organization.GetString("canonified_name") + "/" +
		selection.Wallet.GetString("canonified_name") + "/" + record.GetString("canonified_name")
}

func TestResolveActionAuthorPrerequisites(t *testing.T) {
	app, session, namespace := authorSaveFixture(t)
	onboarding := seedAuthorAction(
		t,
		app,
		session,
		"Onboarding",
		"appId: eu.wallet\n---\n- launchApp\n",
	)
	getPID := seedAuthorAction(t, app, session, "Get PID",
		"appId: eu.wallet\n---\n- openLink: ${deeplink}\n- inputText: ${PIN}\n")
	a := NewResolveActionAuthorPrerequisitesActivity(app)
	literal := func(name string) actionauthor.ParameterSpec {
		return actionauthor.ParameterSpec{
			Name:   name,
			Source: actionauthor.SourceSpec{Kind: actionauthor.SourceLiteral, Value: "x"},
		}
	}

	resolved, err := a.resolve(namespace, ResolveActionAuthorPrerequisitesInput{
		BuildRequest: session,
		Prerequisites: []actionauthor.PrerequisiteSpec{
			{Action: onboarding},
			{
				Action:     getPID,
				Parameters: []actionauthor.ParameterSpec{literal("deeplink"), literal("PIN")},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []ResolvedActionAuthorPrerequisite{
		{
			Action:     onboarding,
			Name:       "Onboarding",
			Code:       "appId: eu.wallet\n---\n- launchApp\n",
			Parameters: []string{},
		},
		{
			Action:     getPID,
			Name:       "Get PID",
			Code:       "appId: eu.wallet\n---\n- openLink: ${deeplink}\n- inputText: ${PIN}\n",
			Parameters: []string{"PIN", "deeplink"},
		},
	}, resolved)

	_, err = a.resolve(namespace, ResolveActionAuthorPrerequisitesInput{
		BuildRequest:  session,
		Prerequisites: []actionauthor.PrerequisiteSpec{{Action: getPID}},
	})
	require.ErrorIs(t, err, actionauthor.ErrInvalid)
	require.ErrorContains(t, err, "needs parameters [PIN deeplink], got []")

	_, err = a.resolve("another-namespace", ResolveActionAuthorPrerequisitesInput{
		BuildRequest:  session,
		Prerequisites: []actionauthor.PrerequisiteSpec{{Action: onboarding}},
	})
	require.ErrorIs(t, err, actionauthor.ErrForbidden)
}

func TestResolveActionAuthorPrerequisitesExecuteMapsErrors(t *testing.T) {
	app, session, _ := authorSaveFixture(t)
	a := NewResolveActionAuthorPrerequisitesActivity(app)

	// The test environment's namespace is not the organization's.
	_, err := executeActivity(t, a, ResolveActionAuthorPrerequisitesInput{BuildRequest: session})
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)

	_, err = executeActivity(t, a, "not a request")
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
}
