// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fcafRunPrefix = "acme/wallet/"

func fcafRunStep(id string, linkedTo string, with map[string]any) pipeline.StepDefinition {
	return pipeline.StepDefinition{
		StepSpec: pipeline.StepSpec{
			ID:   id,
			Use:  StepUseMobileAutomation,
			With: pipeline.StepInputs{Payload: with},
		},
		LinkedTo: linkedTo,
	}
}

func composed(ids ...string) map[string]any {
	actions := make([]any, len(ids))
	for i, id := range ids {
		actions[i] = map[string]any{"id": fcafRunPrefix + id}
	}
	return map[string]any{"actions": actions}
}

// TestFCAFRoleOutcomesOf reads a run where the prelude completed, a composed
// flow failed (CRE315) in its second action, a step linked after a failed
// producer never ran and a step failed for another reason than its flow.
func TestFCAFRoleOutcomesOf(t *testing.T) {
	definition := &pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{
		fcafRunStep("onboard-reference-wallet", "", map[string]any{
			"action_id":  fcafRunPrefix + "fcaf-reset-and-onboard",
			"version_id": fcafRunPrefix + "1-0",
		}),
		{StepSpec: pipeline.StepSpec{ID: "haip-vp-12345678-create", Use: "http-request"}},
		fcafRunStep("haip-vp-12345678-present", "", composed(
			"fcaf-reset-and-onboard", "fcaf-issue-pid", "fcaf-engagement-haip-vp", "fcaf-share",
		)),
		{StepSpec: pipeline.StepSpec{ID: "dc-api-abcdef01-create", Use: "http-request"}},
		fcafRunStep("dc-api-abcdef01-present", "dc-api-abcdef01-create", map[string]any{
			"action_id": fcafRunPrefix + "fcaf-share",
		}),
		fcafRunStep("dc-api-abcdef01-again", "", map[string]any{
			"action_id": fcafRunPrefix + "fcaf-engagement-haip-vp",
		}),
		// Another wallet's action is not this wallet's.
		fcafRunStep("other-step", "", map[string]any{"action_id": "other/wallet/fcaf-share"}),
	}}
	stdout := "Running on emulator-5554\n" +
		" > Flow composed\n" +
		"action " + fcafRunPrefix + "fcaf-reset-and-onboard...\n" +
		"  Launch app \"eu.wallet\"... COMPLETED\n" +
		"action " + fcafRunPrefix + "fcaf-reset-and-onboard... COMPLETED\n" +
		"action " + fcafRunPrefix + "fcaf-issue-pid...\n" +
		"  Tap on \"Accept\"... FAILED\n" +
		"action " + fcafRunPrefix + "fcaf-issue-pid... FAILED\n"
	results := map[string]FCAFStepResult{
		"onboard-reference-wallet": {Completed: true},
		"haip-vp-12345678-create":  {Completed: true},
		"haip-vp-12345678-present": {
			Failed: true,
			Code:   errorcodes.MobileAutomationStepError,
			Stdout: stdout,
		},
		"dc-api-abcdef01-create": {Failed: true, Code: "CRE101"},
		"dc-api-abcdef01-again":  {Failed: true, Code: "CRE228"},
		"other-step":             {Completed: true},
	}
	scenarios := map[string]string{
		"haip-vp-12345678": "fcaf-haip-vp.yaml",
		"dc-api-abcdef01":  "fcaf-dc-api.yaml",
	}

	// The share action's saved role differs from its path's base name.
	roleOf := func(action string) string {
		if action == fcafRunPrefix+"fcaf-share" {
			return "fcaf-z-share"
		}
		return ""
	}
	outcomes := FCAFRoleOutcomesOf(definition, results, scenarios, roleOf)
	require.NotNil(t, outcomes)
	assert.Equal(t, fcafRunPrefix, outcomes.WalletPrefix)
	assert.Equal(t, []FCAFStepOutcome{
		{
			StepID: "onboard-reference-wallet", Action: fcafRunPrefix + "fcaf-reset-and-onboard",
			Role: "fcaf-reset-and-onboard", Status: FCAFRunOK,
		},
		{
			StepID:    "haip-vp-12345678-present",
			Action:    fcafRunPrefix + "fcaf-reset-and-onboard",
			Role:      "fcaf-reset-and-onboard",
			FixtureID: "fcaf-haip-vp.yaml#present",
			Status:    FCAFRunOK,
		},
		{
			StepID: "haip-vp-12345678-present", Action: fcafRunPrefix + "fcaf-issue-pid",
			Role: "fcaf-issue-pid", FixtureID: "fcaf-haip-vp.yaml#present",
			Status: FCAFRunFailing, FailedCommand: `Tap on "Accept"`,
		},
		{
			StepID: "haip-vp-12345678-present", Action: fcafRunPrefix + "fcaf-engagement-haip-vp",
			Role: "fcaf-engagement-haip-vp", FixtureID: "fcaf-haip-vp.yaml#present",
			Status: FCAFRunNotRun,
		},
		{
			StepID: "haip-vp-12345678-present", Action: fcafRunPrefix + "fcaf-share",
			Role: "fcaf-z-share", FixtureID: "fcaf-haip-vp.yaml#present", Status: FCAFRunNotRun,
		},
		{
			StepID: "dc-api-abcdef01-present", Action: fcafRunPrefix + "fcaf-share",
			Role: "fcaf-z-share", FixtureID: "fcaf-dc-api.yaml#present", Status: FCAFRunNotRun,
		},
		{
			StepID: "dc-api-abcdef01-again", Action: fcafRunPrefix + "fcaf-engagement-haip-vp",
			Role: "fcaf-engagement-haip-vp", FixtureID: "fcaf-dc-api.yaml#again",
			Status: FCAFRunNotRun,
		},
	}, outcomes.Steps)
	assert.Equal(t, []FCAFRoleOutcome{
		{
			Action: fcafRunPrefix + "fcaf-engagement-haip-vp",
			Role:   "fcaf-engagement-haip-vp", Status: FCAFRunNotRun, NotRun: 2,
		},
		{
			Action: fcafRunPrefix + "fcaf-issue-pid",
			Role:   "fcaf-issue-pid", Status: FCAFRunFailing, Failing: 1,
			FailedCommand: `Tap on "Accept"`, FixtureID: "fcaf-haip-vp.yaml#present",
			StepID: "haip-vp-12345678-present",
		},
		{
			Action: fcafRunPrefix + "fcaf-reset-and-onboard",
			Role:   "fcaf-reset-and-onboard", Status: FCAFRunOK, OK: 2,
		},
		{
			Action: fcafRunPrefix + "fcaf-share",
			Role:   "fcaf-z-share", Status: FCAFRunNotRun, NotRun: 2,
		},
	}, outcomes.Roles)
}

func TestFCAFRoleOutcomesOfASingleActionAndAnUnlabeledFlow(t *testing.T) {
	definition := &pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{
		fcafRunStep("single", "", map[string]any{"action_id": fcafRunPrefix + "fcaf-share"}),
		fcafRunStep("unlabeled", "", composed("fcaf-issue-pid", "fcaf-share")),
	}}
	failed := FCAFStepResult{
		Failed: true,
		Code:   errorcodes.MobileAutomationStepError,
		Stdout: "Assert that \"Shared\" is visible... FAILED\n",
	}
	outcomes := FCAFRoleOutcomesOf(
		definition,
		map[string]FCAFStepResult{"single": failed, "unlabeled": failed},
		nil,
		nil,
	)
	require.Len(t, outcomes.Steps, 3)
	// A single action owns its flow's failure.
	assert.Equal(t, FCAFRunFailing, outcomes.Steps[0].Status)
	assert.Equal(t, `Assert that "Shared" is visible`, outcomes.Steps[0].FailedCommand)
	// Without a label nothing says which action failed.
	assert.Equal(t, FCAFRunNotRun, outcomes.Steps[1].Status)
	assert.Equal(t, FCAFRunNotRun, outcomes.Steps[2].Status)
	assert.Nil(t, FCAFRoleOutcomesOf(&pipeline.WorkflowDefinition{}, nil, nil, nil))
}

// TestFCAFRoleOutcomesOfARepeatedAction pins that a composed flow running an
// action twice blames the run of it that failed: the labels completed before
// the failed one count the actions that ran.
func TestFCAFRoleOutcomesOfARepeatedAction(t *testing.T) {
	definition := &pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{
		fcafRunStep("present", "", composed("fcaf-issue-pid", "fcaf-issue-pid", "fcaf-share")),
	}}
	stdout := "action " + fcafRunPrefix + "fcaf-issue-pid...\n" +
		"action " + fcafRunPrefix + "fcaf-issue-pid... COMPLETED\n" +
		"action " + fcafRunPrefix + "fcaf-issue-pid...\n" +
		"  Tap on \"Accept\"... FAILED\n" +
		"action " + fcafRunPrefix + "fcaf-issue-pid... FAILED\n"
	outcomes := FCAFRoleOutcomesOf(definition, map[string]FCAFStepResult{"present": {
		Failed: true,
		Code:   errorcodes.MobileAutomationStepError,
		Stdout: stdout,
	}}, nil, nil)
	require.NotNil(t, outcomes)
	statuses := make([]string, 0, len(outcomes.Steps))
	for _, step := range outcomes.Steps {
		statuses = append(statuses, step.Status)
	}
	assert.Equal(t, []string{FCAFRunOK, FCAFRunFailing, FCAFRunNotRun}, statuses)
	assert.Equal(t, `Tap on "Accept"`, outcomes.Steps[1].FailedCommand)
}

// TestFCAFRoleOutcomesOfInjectedSteps pins the fixture of the steps the
// generator injects: a DC API presentation's separate PID issuance and its
// issuer session are the presentation's fixture, and a separate wallet reset
// the fixture of the step it precedes.
func TestFCAFRoleOutcomesOfInjectedSteps(t *testing.T) {
	definition := &pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{
		fcafRunStep("onboard-reference-wallet", "", map[string]any{
			"action_id": fcafRunPrefix + "fcaf-reset-and-onboard",
		}),
		{StepSpec: pipeline.StepSpec{ID: "dc-api-abcdef01-create", Use: "http-request"}},
		{StepSpec: pipeline.StepSpec{
			ID:  "dc-api-abcdef01-present-issue-pid-session",
			Use: "http-request",
		}},
		fcafRunStep("dc-api-abcdef01-present-issue-pid", "", composed(
			"fcaf-reset-and-onboard", "fcaf-issue-pid",
		)),
		{StepSpec: pipeline.StepSpec{
			ID:  "dc-api-abcdef01-present-issue-pid-2-session",
			Use: "http-request",
		}},
		fcafRunStep("dc-api-abcdef01-present-issue-pid-2", "", map[string]any{
			"action_id": fcafRunPrefix + "fcaf-issue-pid",
		}),
		fcafRunStep("dc-api-abcdef01-present", "", map[string]any{
			"action_id": fcafRunPrefix + "fcaf-share",
		}),
		fcafRunStep("browser-12345678-reset-wallet", "", map[string]any{
			"action_id": fcafRunPrefix + "fcaf-reset-and-onboard",
		}),
		fcafRunStep("browser-12345678-open", "", map[string]any{
			"action_id": fcafRunPrefix + "fcaf-share",
		}),
	}}
	results := map[string]FCAFStepResult{}
	for _, step := range definition.Steps {
		results[step.ID] = FCAFStepResult{Completed: true}
	}
	outcomes := FCAFRoleOutcomesOf(definition, results, map[string]string{
		"dc-api-abcdef01":  "fcaf-dc-api.yaml",
		"browser-12345678": "fcaf-browser.yaml",
	}, nil)
	require.NotNil(t, outcomes)
	fixtures := map[string][]string{}
	for _, step := range outcomes.Steps {
		fixtures[step.StepID] = append(fixtures[step.StepID], step.FixtureID)
	}
	assert.Equal(t, map[string][]string{
		"onboard-reference-wallet": {""},
		"dc-api-abcdef01-present-issue-pid": {
			"fcaf-dc-api.yaml#present", "fcaf-dc-api.yaml#present",
		},
		"dc-api-abcdef01-present-issue-pid-2": {"fcaf-dc-api.yaml#present"},
		"dc-api-abcdef01-present":             {"fcaf-dc-api.yaml#present"},
		"browser-12345678-reset-wallet":       {"fcaf-browser.yaml#open"},
		"browser-12345678-open":               {"fcaf-browser.yaml#open"},
	}, fixtures)
}

// attachFCAFRun stores a pipeline result of owner for pipeline, with an
// fcaf_report when report is not nil.
func attachFCAFRun(
	t *testing.T,
	app *tests.TestApp,
	owner string,
	pipelineRecord *core.Record,
	workflowID string,
	report any,
) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	if collection.Fields.GetByName("fcaf_report") == nil {
		collection.Fields.Add(&core.FileField{Name: "fcaf_report", MaxSelect: 1, MaxSize: 1 << 20})
		require.NoError(t, app.Save(collection))
	}
	record := core.NewRecord(collection)
	record.Set("owner", owner)
	record.Set("pipeline", pipelineRecord.Id)
	record.Set("workflow_id", workflowID)
	record.Set("run_id", "run-"+workflowID)
	if report != nil {
		data, err := json.Marshal(report)
		require.NoError(t, err)
		file, err := filesystem.NewFileFromBytes(data, "fcaf-assessment.json")
		require.NoError(t, err)
		record.Set("fcaf_report", []*filesystem.File{file})
	}
	require.NoError(t, app.Save(record))
	// created has millisecond precision: keep the runs apart.
	time.Sleep(5 * time.Millisecond)
}

// fcafRunFixture is a wallet selection of userA's organization with its FCAF
// pipeline records (complete and two subsets) and its current actions of
// three roles.
type fcafRunFixture struct {
	app       *tests.TestApp
	selection Selection
	scope     *FCAFScope
	namespace string
	complete  *core.Record
	subset    *core.Record
	other     *core.Record
	prefix    string
	actions   map[string]string // role → action path
}

func newFCAFRunFixture(t *testing.T) fcafRunFixture {
	t.Helper()
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	return fcafRunFixtureOf(t, app, selection, namespace)
}

func fcafRunFixtureOf(
	t *testing.T,
	app *tests.TestApp,
	selection Selection,
	namespace string,
) fcafRunFixture {
	t.Helper()
	fixture := fcafRunFixture{
		app:       app,
		selection: selection,
		scope:     fcafTestScope(t),
		namespace: namespace,
		prefix:    FCAFWalletPath(selection) + "/",
		actions:   map[string]string{},
	}
	fixture.complete = fixture.pipeline(t, FCAFPipelineName(FCAFWalletPath(selection), nil))
	fixture.subset = fixture.pipeline(t, FCAFPipelineName(FCAFWalletPath(selection),
		[]string{"a.yaml", "b.yaml", "c.yaml"}))
	fixture.other = fixture.pipeline(t, FCAFPipelineName(FCAFWalletPath(selection),
		[]string{"d.yaml", "e.yaml", "f.yaml"}))
	for _, role := range []string{"fcaf-reset-and-onboard", "fcaf-issue-pid", "fcaf-engagement-haip-vp"} {
		fixture.actions[role] = saveFCAFRunAction(t, app, selection.Organization.Id,
			selection.Wallet.Id, fixture.scope, role)
	}
	return fixture
}

// saveFCAFRunAction saves owner's current FCAF action of role on the wallet
// and returns its path.
func saveFCAFRunAction(
	t *testing.T,
	app *tests.TestApp,
	owner, walletID string,
	scope *FCAFScope,
	roleName string,
) string {
	t.Helper()
	role, ok := scope.Contract.Role(roleName)
	require.True(t, ok, roleName)
	collection, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	action := core.NewRecord(collection)
	action.Set("owner", owner)
	action.Set("wallet", walletID)
	action.Set("name", roleName)
	action.Set("code", "appId: eu.wallet\n---\n- launchApp\n")
	action.Set("category", role.Category)
	action.Set(
		"authoring_provenance",
		Provenance{FCAFRole: roleName, FCAFRoleSHA256: role.SHA256()},
	)
	require.NoError(t, app.Save(action))
	path, err := walletActionPath(app, action)
	require.NoError(t, err)
	return path
}

// outcome is how the fixture's action of role did.
func (f fcafRunFixture) outcome(role, status string, counts ...int) FCAFRoleOutcome {
	outcome := FCAFRoleOutcome{Action: f.actions[role], Role: role, Status: status}
	switch status {
	case FCAFRunFailing:
		outcome.Failing = 1
	case FCAFRunOK:
		outcome.OK = 1
	}
	if len(counts) > 0 && status == FCAFRunFailing {
		outcome.Failing = counts[0]
	}
	return outcome
}

func (f fcafRunFixture) pipeline(t *testing.T, name string) *core.Record {
	t.Helper()
	pipelines, err := f.app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	record := core.NewRecord(pipelines)
	record.Set("owner", f.selection.Organization.Id)
	record.Set("name", name)
	record.Set("description", name)
	record.Set("yaml", "name: x\n")
	require.NoError(t, f.app.Save(record))
	return record
}

func (f fcafRunFixture) report(pass int, prefix string, outcomes ...FCAFRoleOutcome) any {
	return map[string]any{
		"summary": map[string]any{"pass": pass, "fail": 2, "blocked": 1},
		"role_outcomes": FCAFRoleOutcomes{
			WalletPrefix: prefix,
			Roles:        outcomes,
			Steps:        []FCAFStepOutcome{},
		},
	}
}

func (f fcafRunFixture) coverage(
	t *testing.T,
	state FCAFRunStateFunc,
) (map[string]FCAFRoleCoverage, *FCAFLastRun) {
	t.Helper()
	coverage, _, lastRun, err := FCAFCoverageWithRun(f.app, f.selection, f.scope, state)
	require.NoError(t, err)
	byRole := map[string]FCAFRoleCoverage{}
	for _, entry := range coverage {
		byRole[entry.Role] = entry
	}
	return byRole, lastRun
}

// TestFCAFCoverageReadsOnlyTheWalletsOwnRuns pins that coverage reads the
// newest finished run of the organization on the wallet's records: another
// organization's row on the same record, a record of another wallet with the
// same display name and a report of another wallet's actions are ignored,
// and a failing run changes neither status nor result.
func TestFCAFCoverageReadsOnlyTheWalletsOwnRuns(t *testing.T) {
	f := newFCAFRunFixture(t)
	orgID := f.selection.Organization.Id
	failing := f.outcome("fcaf-issue-pid", FCAFRunFailing, 2)
	failing.FailedCommand, failing.FixtureID = `Tap on "Accept"`, "f.yaml#present"
	attachFCAFRun(t, f.app, orgID, f.complete, "own", f.report(7, f.prefix,
		f.outcome("fcaf-reset-and-onboard", FCAFRunOK), failing,
	))

	// Another organization queued its own YAML on this record.
	other, err := f.app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	attachFCAFRun(t, f.app, other.Id, f.complete, "intruder", f.report(99, f.prefix))
	// A wallet with the same display name has its own record.
	wallets, err := f.app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	twin := core.NewRecord(wallets)
	twin.Set("owner", orgID)
	twin.Set("name", f.selection.Wallet.GetString("name"))
	require.NoError(t, f.app.Save(twin))
	twinPath := f.selection.WalletOwner.GetString("canonified_name") + "/" +
		twin.GetString("canonified_name")
	require.NotEqual(t, FCAFWalletPath(f.selection), twinPath)
	attachFCAFRun(t, f.app, orgID, f.pipeline(t, FCAFPipelineName(twinPath, nil)), "twin",
		f.report(50, twinPath+"/"))
	// A report of the twin's actions on this wallet's record.
	attachFCAFRun(t, f.app, orgID, f.complete, "foreign-actions", f.report(40, twinPath+"/"))
	// A run still going has no report and is not finished.
	attachFCAFRun(t, f.app, orgID, f.complete, "running", nil)

	byRole, lastRun := f.coverage(t, func(string, string) (FCAFRunState, error) {
		return FCAFRunState{Status: "running"}, nil
	})
	require.NotNil(t, lastRun)
	assert.Equal(t, 7, lastRun.Summary.Pass)
	assert.Equal(t, 10, lastRun.Summary.Total)
	assert.Equal(
		t,
		f.namespace+"/"+f.complete.GetString("canonified_name"),
		lastRun.Summary.Pipeline,
	)
	assert.Contains(t, lastRun.RunURL, "/my/tests/runs/own/run-own")
	require.NotNil(t, byRole["fcaf-issue-pid"].Run)
	assert.Equal(t, FCAFRoleRun{
		Status:        FCAFRunFailing,
		Verdict:       FCAFRunFailing,
		FailedSteps:   2,
		FailedCommand: `Tap on "Accept"`,
		FixtureID:     "f.yaml#present",
		FailedTests:   []string{},
		RunURL:        lastRun.RunURL,
	}, *byRole["fcaf-issue-pid"].Run)
	assert.Equal(t, FCAFRunFailing, byRole["fcaf-issue-pid"].Verdict)
	assert.Equal(t, FCAFVerdictFromPipeline, byRole["fcaf-issue-pid"].VerdictSource)
	assert.Equal(t, FCAFCurrent, byRole["fcaf-issue-pid"].Status)
	assert.Nil(t, byRole["fcaf-engagement-haip-vp"].Run)
	assert.Nil(t, byRole["fcaf-expect-request-rejected"].Run, "a role without an action")
}

// TestFCAFCoverageReadsARunThatEndedWithoutAReport pins that a finished run
// without a report, such as one whose prelude failed, is the last run: every
// role is not run, except the reset role, failing when the prelude failed.
func TestFCAFCoverageReadsARunThatEndedWithoutAReport(t *testing.T) {
	f := newFCAFRunFixture(t)
	orgID := f.selection.Organization.Id
	attachFCAFRun(t, f.app, orgID, f.complete, "older", f.report(7, f.prefix,
		f.outcome("fcaf-issue-pid", FCAFRunOK),
	))
	attachFCAFRun(t, f.app, orgID, f.complete, "prelude-failed", nil)
	attachFCAFRun(t, f.app, orgID, f.complete, "running", nil)
	states := map[string]FCAFRunState{
		"prelude-failed": {
			Finished: true,
			Status:   "failed",
			Failures: map[string]StepFailure{"onboard-reference-wallet": {
				Code:   errorcodes.MobileAutomationStepError,
				Stdout: "Launch app \"eu.wallet\"... FAILED\n",
			}},
		},
		"running": {Status: "running"},
	}
	byRole, lastRun := f.coverage(t, func(workflowID, _ string) (FCAFRunState, error) {
		return states[workflowID], nil
	})
	require.NotNil(t, lastRun)
	assert.Equal(t, "failed", lastRun.Summary.Status)
	assert.Zero(t, lastRun.Summary.Total)
	assert.Contains(t, lastRun.RunURL, "/prelude-failed/")
	assert.Equal(t, FCAFRoleRun{
		Status:        FCAFRunFailing,
		Verdict:       FCAFRunFailing,
		FailedSteps:   1,
		FailedCommand: `Launch app "eu.wallet"`,
		FailedTests:   []string{},
		RunURL:        lastRun.RunURL,
	}, *byRole["fcaf-reset-and-onboard"].Run)
	for name, role := range byRole {
		switch {
		case name == "fcaf-reset-and-onboard":
		case role.Action == "":
			assert.Nil(t, role.Run, name)
		default:
			require.NotNil(t, role.Run, name)
			assert.Equal(t, FCAFRunNotRun, role.Run.Status, name)
			assert.Equal(t, FCAFRunNotRun, role.Run.Verdict, name)
			assert.Empty(t, role.VerdictSource, name, "a role the run did not cover")
		}
	}

	// Without a state reader, the run without a report is left out.
	_, lastRun = f.coverage(t, nil)
	assert.Contains(t, lastRun.RunURL, "/older/")
}

// TestFCAFCoverageKeepsTheCompleteRunUnderNewerSubsets pins that the summary
// comes from the newest complete run and a newer subset run replaces the
// outcome only of the roles it ran.
func TestFCAFCoverageKeepsTheCompleteRunUnderNewerSubsets(t *testing.T) {
	f := newFCAFRunFixture(t)
	orgID := f.selection.Organization.Id
	attachFCAFRun(t, f.app, orgID, f.subset, "subset-old", f.report(1, f.prefix,
		f.outcome("fcaf-engagement-haip-vp", FCAFRunFailing),
	))
	attachFCAFRun(t, f.app, orgID, f.complete, "complete", f.report(20, f.prefix,
		f.outcome("fcaf-reset-and-onboard", FCAFRunOK),
		f.outcome("fcaf-issue-pid", FCAFRunOK),
		f.outcome("fcaf-engagement-haip-vp", FCAFRunOK),
	))
	attachFCAFRun(t, f.app, orgID, f.subset, "subset-new", f.report(2, f.prefix,
		f.outcome("fcaf-issue-pid", FCAFRunFailing),
	))
	// Another, newer subset: both newer subsets count.
	attachFCAFRun(t, f.app, orgID, f.other, "other-new", f.report(3, f.prefix,
		f.outcome("fcaf-reset-and-onboard", FCAFRunFailing),
	))
	byRole, lastRun := f.coverage(t, nil)
	require.NotNil(t, lastRun)
	assert.Equal(t, 20, lastRun.Summary.Pass)
	assert.Contains(t, lastRun.RunURL, "/complete/")
	assert.Equal(t, FCAFRunFailing, byRole["fcaf-issue-pid"].Run.Status)
	assert.Contains(t, byRole["fcaf-issue-pid"].Run.RunURL, "/subset-new/")
	assert.Equal(t, FCAFRunFailing, byRole["fcaf-reset-and-onboard"].Run.Status)
	assert.Contains(t, byRole["fcaf-reset-and-onboard"].Run.RunURL, "/other-new/")
	// The subset older than the complete run changes nothing.
	assert.Equal(t, FCAFRunOK, byRole["fcaf-engagement-haip-vp"].Run.Status)
}

// TestFCAFCoverageMergesSubsetsWithoutACompleteRun pins that, without a
// complete run, every finished subset run counts: roles run only by an older
// subset keep their outcome, a newer subset replaces the outcome of the
// roles it ran, and the summary and link are the newest subset run's.
func TestFCAFCoverageMergesSubsetsWithoutACompleteRun(t *testing.T) {
	f := newFCAFRunFixture(t)
	orgID := f.selection.Organization.Id
	attachFCAFRun(t, f.app, orgID, f.subset, "subset-old", f.report(1, f.prefix,
		f.outcome("fcaf-reset-and-onboard", FCAFRunOK),
		f.outcome("fcaf-engagement-haip-vp", FCAFRunFailing),
	))
	attachFCAFRun(t, f.app, orgID, f.other, "other", f.report(4, f.prefix,
		f.outcome("fcaf-issue-pid", FCAFRunOK),
	))
	byRole, lastRun := f.coverage(t, nil)
	require.NotNil(t, lastRun)
	assert.Equal(t, 4, lastRun.Summary.Pass)
	assert.Contains(t, lastRun.RunURL, "/other/")
	assert.Equal(
		t,
		f.namespace+"/"+f.other.GetString("canonified_name"),
		lastRun.Summary.Pipeline,
	)
	// Disjoint subsets: both outcomes are there.
	require.NotNil(t, byRole["fcaf-issue-pid"].Run)
	assert.Equal(t, FCAFRunOK, byRole["fcaf-issue-pid"].Run.Status)
	assert.Contains(t, byRole["fcaf-issue-pid"].Run.RunURL, "/other/")
	require.NotNil(t, byRole["fcaf-engagement-haip-vp"].Run)
	assert.Equal(t, FCAFRunFailing, byRole["fcaf-engagement-haip-vp"].Run.Status)
	assert.Contains(t, byRole["fcaf-engagement-haip-vp"].Run.RunURL, "/subset-old/")

	// A newer subset replaces the outcome of the roles it ran only.
	attachFCAFRun(t, f.app, orgID, f.subset, "subset-new", f.report(2, f.prefix,
		f.outcome("fcaf-engagement-haip-vp", FCAFRunOK),
	))
	byRole, lastRun = f.coverage(t, nil)
	assert.Equal(t, 2, lastRun.Summary.Pass)
	assert.Contains(t, lastRun.RunURL, "/subset-new/")
	assert.Equal(t, FCAFRunOK, byRole["fcaf-engagement-haip-vp"].Run.Status)
	assert.Contains(t, byRole["fcaf-engagement-haip-vp"].Run.RunURL, "/subset-new/")
	assert.Equal(t, FCAFRunOK, byRole["fcaf-reset-and-onboard"].Run.Status)
	assert.Contains(t, byRole["fcaf-reset-and-onboard"].Run.RunURL, "/subset-old/")
	assert.Contains(t, byRole["fcaf-issue-pid"].Run.RunURL, "/other/")
}

// TestFCAFCoverageJudgesRolesOnTheirTests pins the pipeline verdicts: a
// tolerant role whose action ran while one of its tests failed is a finding,
// a failing action is failing, a role whose tests all passed (skipped ones
// aside) passed, one with a blocked test is blocked, and a role the run did
// not cover keeps the agent's verdict as not verified yet. A newer subset
// replaces the status of the tests it ran.
func TestFCAFCoverageJudgesRolesOnTheirTests(t *testing.T) {
	f := newFCAFRunFixture(t)
	orgID := f.selection.Organization.Id
	roleTests, err := f.scope.Contract.RoleTests(f.scope.Root, f.scope.Suite)
	require.NoError(t, err)
	const tolerant, blocked, unverified = "fcaf-exercise-wallet-error", "fcaf-observe-request",
		"fcaf-expect-no-matching-document"
	for _, role := range []string{tolerant, blocked, unverified} {
		f.actions[role] = saveFCAFRunAction(t, f.app, orgID, f.selection.Wallet.Id, f.scope, role)
		require.NotEmpty(t, roleTests[role], role)
	}
	record, err := f.app.FindFirstRecordByFilter("wallet_actions", "owner = {:o} && name = {:n}",
		map[string]any{"o": orgID, "n": unverified})
	require.NoError(t, err)
	role, _ := f.scope.Contract.Role(unverified)
	record.Set("authoring_provenance", Provenance{
		FCAFRole:       unverified,
		FCAFRoleSHA256: role.SHA256(),
		FCAFResult:     &FCAFResult{Status: FCAFReplayPassed, Note: "shared nothing"},
	})
	require.NoError(t, f.app.Save(record))

	executed := []map[string]any{}
	add := func(status string, ids ...string) {
		for _, id := range ids {
			executed = append(executed, map[string]any{"test_id": id, "status": status})
		}
	}
	haip := roleTests["fcaf-engagement-haip-vp"]
	add("passed", haip[1:]...)
	add("skipped", haip[0])
	add("failed", roleTests[tolerant][0])
	add("passed", roleTests[tolerant][1:]...)
	add("blocked", roleTests[blocked][0])
	add("passed", roleTests[blocked][1:]...)
	add("failed", roleTests[unverified]...)
	report := func(pass int, outcomes ...FCAFRoleOutcome) map[string]any {
		r, _ := f.report(pass, f.prefix, outcomes...).(map[string]any)
		r["executed_tests"] = executed
		return r
	}
	attachFCAFRun(t, f.app, orgID, f.complete, "complete", report(9,
		f.outcome("fcaf-reset-and-onboard", FCAFRunOK),
		f.outcome("fcaf-issue-pid", FCAFRunFailing),
		f.outcome("fcaf-engagement-haip-vp", FCAFRunOK),
		f.outcome(tolerant, FCAFRunOK),
		f.outcome(blocked, FCAFRunOK),
	))
	byRole, _ := f.coverage(t, nil)

	finding := byRole[tolerant]
	require.NotNil(t, finding.Run)
	assert.Equal(t, FCAFRunOK, finding.Run.Status)
	assert.Equal(t, FCAFVerdictFinding, finding.Run.Verdict)
	assert.Equal(t, []string{roleTests[tolerant][0]}, finding.Run.FailedTests)
	assert.Equal(t, len(roleTests[tolerant]), finding.Run.TestsRun)
	assert.Equal(t, FCAFVerdictFinding, finding.Verdict)
	assert.Equal(t, FCAFVerdictFromPipeline, finding.VerdictSource)

	assert.Equal(t, FCAFRunFailing, byRole["fcaf-issue-pid"].Verdict)
	assert.Equal(t, FCAFVerdictPassed, byRole["fcaf-reset-and-onboard"].Verdict, "no tests")
	passed := byRole["fcaf-engagement-haip-vp"]
	assert.Equal(t, FCAFVerdictPassed, passed.Verdict)
	assert.Equal(t, len(haip)-1, passed.Run.TestsRun, "a skipped test does not count")
	assert.Empty(t, passed.Run.FailedTests)
	assert.Equal(t, FCAFVerdictBlocked, byRole[blocked].Verdict)

	notVerified := byRole[unverified]
	assert.Nil(t, notVerified.Run)
	assert.Equal(t, FCAFReplayPassed, notVerified.Verdict)
	assert.Equal(t, FCAFVerdictFromAuthoring, notVerified.VerdictSource)
	assert.Equal(t, "shared nothing", notVerified.Note)

	// A newer subset reran the tolerant role: its test now passes.
	executed = nil
	add("passed", roleTests[tolerant][0])
	attachFCAFRun(t, f.app, orgID, f.subset, "subset", report(1, f.outcome(tolerant, FCAFRunOK)))
	byRole, _ = f.coverage(t, nil)
	assert.Equal(t, FCAFVerdictPassed, byRole[tolerant].Verdict)
	assert.Equal(t, len(roleTests[tolerant]), byRole[tolerant].Run.TestsRun)
	assert.Equal(t, FCAFVerdictBlocked, byRole[blocked].Verdict, "kept from the complete run")

	// The blocked role's action is saved again after the runs: they tested
	// older code, so its verdict is the agent's until a run covers it again.
	record, err = f.app.FindFirstRecordByFilter("wallet_actions", "owner = {:o} && name = {:n}",
		map[string]any{"o": orgID, "n": blocked})
	require.NoError(t, err)
	role, _ = f.scope.Contract.Role(blocked)
	record.Set("authoring_provenance", Provenance{
		FCAFRole:       blocked,
		FCAFRoleSHA256: role.SHA256(),
		FCAFResult:     &FCAFResult{Status: FCAFReplayPassed, Note: "rewritten"},
	})
	require.NoError(t, f.app.Save(record))
	byRole, _ = f.coverage(t, nil)
	rewritten := byRole[blocked]
	require.NotNil(t, rewritten.Run, "the run's outcome stays visible")
	assert.Equal(t, FCAFVerdictBlocked, rewritten.Run.Verdict)
	assert.Equal(t, FCAFReplayPassed, rewritten.Verdict)
	assert.Equal(t, FCAFVerdictFromAuthoring, rewritten.VerdictSource)
	assert.Equal(t, FCAFVerdictFromPipeline, byRole[tolerant].VerdictSource)
}

func TestFCAFRoleRunJudgeBoundsFailedTests(t *testing.T) {
	tests := make([]string, 30)
	statuses := map[string]string{}
	for i := range tests {
		tests[i] = fmt.Sprintf("T%02d", i)
		statuses[tests[i]] = "failed"
	}
	run := FCAFRoleRun{Status: FCAFRunOK}
	run.judge(tests, statuses)
	assert.Equal(t, FCAFVerdictFinding, run.Verdict)
	assert.Len(t, run.FailedTests, MaxFCAFFailedTests)
	assert.Equal(t, 30, run.TestsRun)
}

// TestFCAFCoverageReadsOnlyTheNewestResults pins that coverage reads only
// the fcafRunScanLimit newest results, asking Temporal about each at most
// once: a finished run older than them is not read.
func TestFCAFCoverageReadsOnlyTheNewestResults(t *testing.T) {
	f := newFCAFRunFixture(t)
	orgID := f.selection.Organization.Id
	attachFCAFRun(t, f.app, orgID, f.complete, "old", f.report(7, f.prefix,
		f.outcome("fcaf-issue-pid", FCAFRunOK),
	))
	for i := range fcafRunScanLimit {
		attachFCAFRun(t, f.app, orgID, f.subset, fmt.Sprintf("running-%d", i), nil)
	}
	asked := map[string]int{}
	byRole, lastRun := f.coverage(t, func(workflowID, _ string) (FCAFRunState, error) {
		asked[workflowID]++
		return FCAFRunState{Status: "running"}, nil
	})
	assert.Nil(t, lastRun)
	assert.Nil(t, byRole["fcaf-issue-pid"].Run)
	assert.Len(t, asked, fcafRunScanLimit)
	for workflowID, count := range asked {
		assert.Equal(t, 1, count, workflowID)
	}
}

// TestFCAFPipelineName pins the record names: the complete one, and a
// subset's named by its set of scenarios whatever their order.
func TestFCAFPipelineName(t *testing.T) {
	complete := FCAFPipelineName("org/wallet", nil)
	assert.Equal(t, "FCAF relying-party validation - org/wallet", complete)
	subset := FCAFPipelineName("org/wallet", []string{"b.yaml", "a.yaml"})
	assert.Regexp(t, `^`+regexp.QuoteMeta(complete)+` - scenarios [0-9a-f]{8}$`, subset)
	assert.Equal(t, subset, FCAFPipelineName("org/wallet", []string{"a.yaml", "b.yaml"}))
	assert.NotEqual(t, subset, FCAFPipelineName("org/wallet", []string{"a.yaml", "c.yaml"}))
	sum := sha256.Sum256([]byte("a.yaml\nb.yaml"))
	assert.Equal(t, complete+" - scenarios "+hex.EncodeToString(sum[:4]), subset)
}

// TestFCAFCoverageMatchesRunsByActionPath reads back a run of the caller's
// actions on another organization's published wallet: they live under the
// wallet owner's path with a suffix (the owner has actions of the same
// names), and coverage matches the run's outcomes by that path.
func TestFCAFCoverageMatchesRunsByActionPath(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	owner, err := app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", owner.Id)
	wallet.Set("name", "Published FCAF wallet")
	wallet.Set("published", true)
	require.NoError(t, app.Save(wallet))
	scope := fcafTestScope(t)
	ownersPath := saveFCAFRunAction(t, app, owner.Id, wallet.Id, scope, "fcaf-issue-pid")
	selection, err := SelectWallet(
		app,
		namespace,
		request.Actor,
		owner.GetString("canonified_name")+"/"+wallet.GetString("canonified_name"),
	)
	require.NoError(t, err)
	f := fcafRunFixtureOf(t, app, selection, namespace)
	require.NotEqual(t, ownersPath, f.actions["fcaf-issue-pid"])
	require.Equal(t, path.Dir(ownersPath), path.Dir(f.actions["fcaf-issue-pid"]))
	require.NotEqual(t, "fcaf-issue-pid", path.Base(f.actions["fcaf-issue-pid"]))

	attachFCAFRun(t, app, selection.Organization.Id, f.complete, "foreign", f.report(3, f.prefix,
		f.outcome("fcaf-issue-pid", FCAFRunFailing),
		// The owner's action of the same role is not the caller's.
		FCAFRoleOutcome{Action: ownersPath, Role: "fcaf-issue-pid", Status: FCAFRunOK, OK: 1},
	))
	byRole, lastRun := f.coverage(t, nil)
	require.NotNil(t, lastRun)
	require.Equal(t, f.actions["fcaf-issue-pid"], byRole["fcaf-issue-pid"].Action)
	require.NotNil(t, byRole["fcaf-issue-pid"].Run)
	assert.Equal(t, FCAFRunFailing, byRole["fcaf-issue-pid"].Run.Status)
	assert.Nil(t, byRole["fcaf-reset-and-onboard"].Run)
}

func TestFCAFWrittenActions(t *testing.T) {
	assert.Equal(t, map[string]string{"a": "o/w/a", "b": "o/w/b-1"}, FCAFWrittenActions(
		[]FCAFRoleCoverage{
			{Role: "a", Status: FCAFCurrent, Action: "o/w/a"},
			{Role: "b", Status: FCAFIncomplete, Action: "o/w/b-1"},
			{Role: "c", Status: FCAFStale, Action: "o/w/c"},
			{Role: "d", Status: FCAFMissing},
			{Role: "e", Status: FCAFUnmanaged, Action: "o/w/e"},
		},
	))
}
