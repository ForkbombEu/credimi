// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"encoding/json"
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

	outcomes := FCAFRoleOutcomesOf(definition, results, scenarios)
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
			Role: "fcaf-share", FixtureID: "fcaf-haip-vp.yaml#present", Status: FCAFRunNotRun,
		},
		{
			StepID: "dc-api-abcdef01-present", Action: fcafRunPrefix + "fcaf-share",
			Role: "fcaf-share", FixtureID: "fcaf-dc-api.yaml#present", Status: FCAFRunNotRun,
		},
		{
			StepID: "dc-api-abcdef01-again", Action: fcafRunPrefix + "fcaf-engagement-haip-vp",
			Role: "fcaf-engagement-haip-vp", FixtureID: "fcaf-dc-api.yaml#again",
			Status: FCAFRunNotRun,
		},
	}, outcomes.Steps)
	assert.Equal(t, []FCAFRoleOutcome{
		{Role: "fcaf-engagement-haip-vp", Status: FCAFRunNotRun, NotRun: 2},
		{
			Role: "fcaf-issue-pid", Status: FCAFRunFailing, Failing: 1,
			FailedCommand: `Tap on "Accept"`, FixtureID: "fcaf-haip-vp.yaml#present",
			StepID: "haip-vp-12345678-present",
		},
		{Role: "fcaf-reset-and-onboard", Status: FCAFRunOK, OK: 2},
		{Role: "fcaf-share", Status: FCAFRunNotRun, NotRun: 2},
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
	)
	require.Len(t, outcomes.Steps, 3)
	// A single action owns its flow's failure.
	assert.Equal(t, FCAFRunFailing, outcomes.Steps[0].Status)
	assert.Equal(t, `Assert that "Shared" is visible`, outcomes.Steps[0].FailedCommand)
	// Without a label nothing says which action failed.
	assert.Equal(t, FCAFRunNotRun, outcomes.Steps[1].Status)
	assert.Equal(t, FCAFRunNotRun, outcomes.Steps[2].Status)
	assert.Nil(t, FCAFRoleOutcomesOf(&pipeline.WorkflowDefinition{}, nil, nil))
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
// pipeline records.
type fcafRunFixture struct {
	app       *tests.TestApp
	selection Selection
	scope     *FCAFScope
	namespace string
	complete  *core.Record
	subset    *core.Record
	prefix    string
}

func newFCAFRunFixture(t *testing.T) fcafRunFixture {
	t.Helper()
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	fixture := fcafRunFixture{
		app:       app,
		selection: selection,
		scope:     fcafTestScope(t),
		namespace: namespace,
		prefix:    FCAFWalletPath(selection) + "/",
	}
	fixture.complete = fixture.pipeline(t, FCAFPipelineName(FCAFWalletPath(selection), 0))
	fixture.subset = fixture.pipeline(t, FCAFPipelineName(FCAFWalletPath(selection), 3))
	return fixture
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
	coverage, _, lastRun, err := FCAFCoverage(f.app, f.selection, f.scope, state)
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
	attachFCAFRun(t, f.app, orgID, f.complete, "own", f.report(7, f.prefix,
		FCAFRoleOutcome{Role: "fcaf-reset-and-onboard", Status: FCAFRunOK, OK: 3},
		FCAFRoleOutcome{
			Role: "fcaf-issue-pid", Status: FCAFRunFailing, Failing: 2, OK: 1,
			FailedCommand: `Tap on "Accept"`, FixtureID: "f.yaml#present",
		},
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
	attachFCAFRun(t, f.app, orgID, f.pipeline(t, FCAFPipelineName(twinPath, 0)), "twin",
		f.report(50, twinPath+"/"))
	// A report of the twin's actions on this wallet's record.
	attachFCAFRun(t, f.app, orgID, f.complete, "foreign-actions", f.report(40, twinPath+"/"))
	// A run still going has no report and is not finished.
	attachFCAFRun(t, f.app, orgID, f.complete, "running", nil)

	byRole, lastRun := f.coverage(t, func(string, string) (FCAFRunState, error) {
		return FCAFRunState{Status: "Running"}, nil
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
		FailedSteps:   2,
		FailedCommand: `Tap on "Accept"`,
		FixtureID:     "f.yaml#present",
		RunURL:        lastRun.RunURL,
	}, *byRole["fcaf-issue-pid"].Run)
	assert.Equal(t, FCAFMissing, byRole["fcaf-issue-pid"].Status)
	assert.Nil(t, byRole["fcaf-engagement-haip-vp"].Run)
}

// TestFCAFCoverageReadsARunThatEndedWithoutAReport pins that a finished run
// without a report, such as one whose prelude failed, is the last run: every
// role is not run, except the reset role, failing when the prelude failed.
func TestFCAFCoverageReadsARunThatEndedWithoutAReport(t *testing.T) {
	f := newFCAFRunFixture(t)
	orgID := f.selection.Organization.Id
	attachFCAFRun(t, f.app, orgID, f.complete, "older", f.report(7, f.prefix,
		FCAFRoleOutcome{Role: "fcaf-issue-pid", Status: FCAFRunOK, OK: 1},
	))
	attachFCAFRun(t, f.app, orgID, f.complete, "prelude-failed", nil)
	attachFCAFRun(t, f.app, orgID, f.complete, "running", nil)
	states := map[string]FCAFRunState{
		"prelude-failed": {
			Finished: true,
			Status:   "Failed",
			Failures: map[string]StepFailure{"onboard-reference-wallet": {
				Code:   errorcodes.MobileAutomationStepError,
				Stdout: "Launch app \"eu.wallet\"... FAILED\n",
			}},
		},
		"running": {Status: "Running"},
	}
	byRole, lastRun := f.coverage(t, func(workflowID, _ string) (FCAFRunState, error) {
		return states[workflowID], nil
	})
	require.NotNil(t, lastRun)
	assert.Equal(t, "Failed", lastRun.Summary.Status)
	assert.Zero(t, lastRun.Summary.Total)
	assert.Contains(t, lastRun.RunURL, "/prelude-failed/")
	assert.Equal(t, FCAFRoleRun{
		Status:        FCAFRunFailing,
		FailedSteps:   1,
		FailedCommand: `Launch app "eu.wallet"`,
		RunURL:        lastRun.RunURL,
	}, *byRole["fcaf-reset-and-onboard"].Run)
	for name, role := range byRole {
		if name != "fcaf-reset-and-onboard" {
			require.NotNil(t, role.Run, name)
			assert.Equal(t, FCAFRunNotRun, role.Run.Status, name)
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
		FCAFRoleOutcome{Role: "fcaf-engagement-haip-vp", Status: FCAFRunFailing, Failing: 1},
	))
	attachFCAFRun(t, f.app, orgID, f.complete, "complete", f.report(20, f.prefix,
		FCAFRoleOutcome{Role: "fcaf-reset-and-onboard", Status: FCAFRunOK, OK: 9},
		FCAFRoleOutcome{Role: "fcaf-issue-pid", Status: FCAFRunOK, OK: 9},
		FCAFRoleOutcome{Role: "fcaf-engagement-haip-vp", Status: FCAFRunOK, OK: 1},
	))
	attachFCAFRun(t, f.app, orgID, f.subset, "subset-new", f.report(2, f.prefix,
		FCAFRoleOutcome{
			Role: "fcaf-issue-pid", Status: FCAFRunFailing, Failing: 1,
			FailedCommand: `Tap on "Accept"`,
		},
	))
	byRole, lastRun := f.coverage(t, nil)
	require.NotNil(t, lastRun)
	assert.Equal(t, 20, lastRun.Summary.Pass)
	assert.Contains(t, lastRun.RunURL, "/complete/")
	assert.Equal(t, FCAFRunFailing, byRole["fcaf-issue-pid"].Run.Status)
	assert.Contains(t, byRole["fcaf-issue-pid"].Run.RunURL, "/subset-new/")
	assert.Equal(t, FCAFRunOK, byRole["fcaf-reset-and-onboard"].Run.Status)
	// The subset older than the complete run changes nothing.
	assert.Equal(t, FCAFRunOK, byRole["fcaf-engagement-haip-vp"].Run.Status)
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
