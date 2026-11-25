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

// attachFCAFRun stores a pipeline result of pipeline with an fcaf_report.
func attachFCAFRun(
	t *testing.T,
	app *tests.TestApp,
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
	record.Set("owner", pipelineRecord.GetString("owner"))
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

// TestFCAFCoverageMergesTheLatestRun pins that coverage reads the newest run
// with a stored report among the wallet's FCAF pipelines, complete or
// subset, and that a failing run changes neither status nor result.
func TestFCAFCoverageMergesTheLatestRun(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	scope := fcafTestScope(t)
	pipelines, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	newPipeline := func(name string) *core.Record {
		record := core.NewRecord(pipelines)
		record.Set("owner", selection.Organization.Id)
		record.Set("name", name)
		record.Set("description", name)
		record.Set("yaml", "name: x\n")
		require.NoError(t, app.Save(record))
		return record
	}
	walletName := selection.Wallet.GetString("name")
	complete := newPipeline(FCAFPipelineName(walletName, 0))
	subset := newPipeline(FCAFPipelineName(walletName, 3))
	unrelated := newPipeline(FCAFPipelineName(walletName+" two", 0))

	report := func(pass int, outcomes []FCAFRoleOutcome) map[string]any {
		return map[string]any{
			"summary": map[string]any{"pass": pass, "fail": 2, "blocked": 1},
			"role_outcomes": FCAFRoleOutcomes{
				WalletPrefix: "x/y/",
				Roles:        outcomes,
				Steps:        []FCAFStepOutcome{},
			},
		}
	}
	attachFCAFRun(t, app, complete, "old", report(1, []FCAFRoleOutcome{
		{Role: "fcaf-issue-pid", Status: FCAFRunOK, OK: 1},
	}))
	attachFCAFRun(t, app, subset, "latest", report(7, []FCAFRoleOutcome{
		{Role: "fcaf-reset-and-onboard", Status: FCAFRunOK, OK: 3},
		{
			Role: "fcaf-issue-pid", Status: FCAFRunFailing, Failing: 2, OK: 1,
			FailedCommand: `Tap on "Accept"`, FixtureID: "f.yaml#present",
		},
	}))
	// A run still going (no report yet) and another wallet's run are not
	// the wallet's last finished run.
	attachFCAFRun(t, app, complete, "running", nil)
	attachFCAFRun(t, app, unrelated, "other", report(0, nil))

	coverage, _, lastRun, err := FCAFCoverage(app, selection, scope)
	require.NoError(t, err)
	require.NotNil(t, lastRun)
	assert.Equal(t, 7, lastRun.Summary.Pass)
	assert.Equal(t, 10, lastRun.Summary.Total)
	assert.Equal(t, namespace+"/"+subset.GetString("canonified_name"), lastRun.Summary.Pipeline)
	assert.Contains(t, lastRun.RunURL, "/my/tests/runs/latest/run-latest")
	byRole := map[string]FCAFRoleCoverage{}
	for _, entry := range coverage {
		byRole[entry.Role] = entry
	}
	require.NotNil(t, byRole["fcaf-issue-pid"].Run)
	assert.Equal(t, FCAFRoleRun{
		Status:        FCAFRunFailing,
		FailedSteps:   2,
		FailedCommand: `Tap on "Accept"`,
		FixtureID:     "f.yaml#present",
		RunURL:        lastRun.RunURL,
	}, *byRole["fcaf-issue-pid"].Run)
	assert.Equal(t, FCAFMissing, byRole["fcaf-issue-pid"].Status)
	assert.Equal(t, FCAFRunOK, byRole["fcaf-reset-and-onboard"].Run.Status)
	assert.Nil(t, byRole["fcaf-engagement-haip-vp"].Run)
}

func TestFCAFWrittenRoles(t *testing.T) {
	assert.Equal(t, map[string]bool{
		"a": true, "b": true, "c": false, "d": false, "e": false,
	}, FCAFWrittenRoles([]FCAFRoleCoverage{
		{Role: "a", Status: FCAFCurrent},
		{Role: "b", Status: FCAFIncomplete},
		{Role: "c", Status: FCAFStale},
		{Role: "d", Status: FCAFMissing},
		{Role: "e", Status: FCAFUnmanaged},
	}))
}
