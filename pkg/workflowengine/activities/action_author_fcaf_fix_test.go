// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

// fcafFixRole is the role the fix session tests make fail in the last run.
const fcafFixRole = "fcaf-expect-request-rejected"

// fcafFixSessionLookup fakes a running, Credimi-signed FCAF fix session of
// the relying-party suite on session's build.
func fcafFixSessionLookup(
	t *testing.T,
	app core.App,
	session actionauthor.BuildRequest,
) sessionLookup {
	t.Helper()
	lookup := unsignedSessionLookup(app)
	lookup.history = signedSessionHistory(t, app, map[string]any{
		"platform":                  session.Platform,
		"wallet_version_identifier": session.WalletVersionIdentifier,
		"actor_id":                  session.ActorID,
		"mode":                      actionauthor.SessionModeFCAF,
		"fcaf_suite":                actionauthor.FCAFRelyingPartySuite,
		"fcaf_fix":                  true,
	})
	return lookup
}

// seedFCAFFixRun saves the session wallet's current action of fcafFixRole
// and a finished run of the wallet's FCAF pipeline in which it failed on a
// fixture of the role outside its authoring set, which it returns.
func seedFCAFFixRun(
	t *testing.T,
	app core.App,
	session actionauthor.BuildRequest,
) walletroles.Fixture {
	t.Helper()
	scope := testFCAFScope(t)
	role, ok := scope.Contract.Role(fcafFixRole)
	require.True(t, ok)
	set, err := scope.Contract.AuthoringSet(scope.Root, scope.Suite, fcafFixRole)
	require.NoError(t, err)
	all, err := scope.Contract.Fixtures(scope.Root, scope.Suite, fcafFixRole)
	require.NoError(t, err)
	index := slices.IndexFunc(all, func(fixture walletroles.Fixture) bool {
		return !slices.ContainsFunc(set, func(s walletroles.Fixture) bool {
			return s.ID == fixture.ID
		})
	})
	require.GreaterOrEqual(t, index, 0, "the role has a fixture outside its authoring set")
	failed := all[index]

	action := seedAuthorAction(t, app, session, fcafFixRole, "appId: eu.wallet")
	recordProvenance(t, app, action, actionauthor.Provenance{
		FCAFRole:       fcafFixRole,
		FCAFRoleSHA256: role.SHA256(),
	})
	selection, err := actionauthor.Select(app, "", session)
	require.NoError(t, err)
	walletPath := actionauthor.FCAFWalletPath(selection)
	pipelines, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelines)
	pipelineRecord.Set("owner", selection.Organization.Id)
	pipelineRecord.Set("name", actionauthor.FCAFPipelineName(walletPath, nil))
	pipelineRecord.Set("description", "fcaf")
	pipelineRecord.Set("yaml", "name: x\n")
	require.NoError(t, app.Save(pipelineRecord))

	ensureFCAFReportFields(t, app)
	reported := strings.Replace(failed.ID, "#", ".yaml#", 1)
	report, err := json.Marshal(map[string]any{
		"summary": map[string]any{"pass": 1, "fail": 1},
		"role_outcomes": actionauthor.FCAFRoleOutcomes{
			WalletPrefix: walletPath + "/",
			Roles: []actionauthor.FCAFRoleOutcome{{
				Action:        action,
				Role:          fcafFixRole,
				Status:        actionauthor.FCAFRunFailing,
				Failing:       1,
				FailedCommand: `Assert that "Request rejected" is visible`,
				FixtureID:     reported,
			}},
			Steps: []actionauthor.FCAFStepOutcome{{
				StepID:    "s1",
				Action:    action,
				Role:      fcafFixRole,
				FixtureID: reported,
				Status:    actionauthor.FCAFRunFailing,
			}},
		},
	})
	require.NoError(t, err)
	results, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	result := core.NewRecord(results)
	result.Set("owner", selection.Organization.Id)
	result.Set("pipeline", pipelineRecord.Id)
	result.Set("workflow_id", "Pipeline-fix")
	result.Set("run_id", "run-fix")
	file, err := filesystem.NewFileFromBytes(report, "fcaf-assessment.json")
	require.NoError(t, err)
	result.Set("fcaf_report", []*filesystem.File{file})
	require.NoError(t, app.Save(result))
	return failed
}

func TestFCAFFixSessionListsAndDescribesTheFailedFixtures(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, _ := authorSaveFixture(t)
	failed := seedFCAFFixRun(t, app, session)
	caller := authorTestCaller()

	list := NewListActionAuthorFCAFRolesActivity(app)
	list.authorSessionLookup = fcafFixSessionLookup(t, app, session).lookup()
	output, err := list.list(t.Context(), caller)
	require.NoError(t, err)
	fixed := []string{}
	for _, role := range output.Roles {
		if role.Fix {
			fixed = append(fixed, role.Role)
			require.Equal(t, []string{failed.ID}, role.FixFixtures)
			require.Equal(t, `Assert that "Request rejected" is visible`, role.FailedCommand)
		}
	}
	require.Equal(t, []string{fcafFixRole}, fixed)

	describe := func(lookup sessionLookup) DescribeActionAuthorFCAFRoleOutput {
		a := NewDescribeActionAuthorFCAFRoleActivity(app)
		a.authorSessionLookup = lookup.lookup()
		described, err := a.describe(t.Context(), caller, DescribeActionAuthorFCAFRoleInput{
			Role: fcafFixRole,
		})
		require.NoError(t, err)
		return described
	}
	withFix := describe(fcafFixSessionLookup(t, app, session))
	last := withFix.Fixtures[len(withFix.Fixtures)-1]
	require.Equal(t, failed.ID, last.ID)
	require.True(t, last.Fix)
	require.Equal(t, failed.Class, last.Class)
	for _, fixture := range withFix.Fixtures[:len(withFix.Fixtures)-1] {
		require.False(t, fixture.Fix, fixture.ID)
	}
	// An authoring session describes only the authoring set.
	authoring := describe(fcafSessionLookup(t, app, session))
	require.Len(t, authoring.Fixtures, len(withFix.Fixtures)-1)
	for _, fixture := range authoring.Fixtures {
		require.NotEqual(t, failed.ID, fixture.ID)
	}

	// The replays of a fix session may cover the failed fixture.
	ref := ActionAuthorSessionRef{
		Mode:      actionauthor.SessionModeFCAF,
		FCAFSuite: actionauthor.FCAFRelyingPartySuite,
	}
	selection, err := actionauthor.Select(app, "", session)
	require.NoError(t, err)
	replays := []actionauthor.FCAFReplay{{
		Fixture: failed.ID, Class: failed.Class, Status: actionauthor.FCAFReplayPassed,
	}}
	scope, err := FCAFFixScope(app, ref, selection, testFCAFScope(t), nil)
	require.NoError(t, err)
	require.ErrorIs(
		t,
		actionauthor.CheckFCAFReplays(scope, replays, fcafFixRole),
		actionauthor.ErrInvalid,
	)
	ref.FCAFFix = true
	scope, err = FCAFFixScope(app, ref, selection, testFCAFScope(t), nil)
	require.NoError(t, err)
	require.Equal(t, map[string][]string{fcafFixRole: {failed.ID}}, scope.Fix)
	require.NoError(t, actionauthor.CheckFCAFReplays(scope, replays, fcafFixRole))
}

func TestFCAFFixFlagIsSignedByCredimi(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, _ := authorSaveFixture(t)
	organization, _, _ := strings.Cut(session.WalletVersionIdentifier, "/")
	// Credimi signed an authoring session; its start input claims a fix.
	grant, err := actionauthor.SignSession(app, actionauthor.SessionGrant{
		SessionWorkflowID:       authorTestSessionID,
		Namespace:               authorTestNamespace,
		Organization:            organization,
		Platform:                session.Platform,
		WalletVersionIdentifier: session.WalletVersionIdentifier,
		ActorID:                 session.ActorID,
		Mode:                    actionauthor.SessionModeFCAF,
		FCAFSuite:               actionauthor.FCAFRelyingPartySuite,
	})
	require.NoError(t, err)
	// The grant check refuses the start input before any run grant is read.
	lookup := unsignedSessionLookup(app)
	lookup.history = sessionHistory(t, map[string]any{
		"platform":                  session.Platform,
		"wallet_version_identifier": session.WalletVersionIdentifier,
		"actor_id":                  session.ActorID,
		"mode":                      actionauthor.SessionModeFCAF,
		"fcaf_suite":                actionauthor.FCAFRelyingPartySuite,
		"fcaf_fix":                  true,
		"namespace":                 authorTestNamespace,
		"organization":              organization,
		"grant":                     grant,
	})
	list := NewListActionAuthorFCAFRolesActivity(app)
	list.authorSessionLookup = lookup.lookup()

	_, err = list.list(t.Context(), authorTestCaller())
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "action author session was not started by Credimi")
}
