// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// How a wallet action did in an FCAF pipeline run.
const (
	// FCAFRunOK: the step running the action completed, or failed in a later
	// action of its flow.
	FCAFRunOK = "ok"
	// FCAFRunFailing: the step failed (CRE315) inside this action.
	FCAFRunFailing = "failing"
	// FCAFRunNotRun: the step did not run, failed before or after this
	// action, or failed for another reason than its Maestro flow.
	FCAFRunNotRun = "not_run"
)

// fcafPipelineNamePrefix names a wallet's FCAF relying-party pipeline.
const fcafPipelineNamePrefix = "FCAF relying-party validation - "

// FCAFPipelineName is the name of the pipeline record that runs the FCAF
// relying-party validation of a wallet: one per wallet, and one per number
// of scenarios for a subset (scenarios above 0), so a subset never replaces
// the wallet's complete pipeline.
func FCAFPipelineName(walletName string, scenarios int) string {
	name := fcafPipelineNamePrefix + walletName
	if scenarios > 0 {
		name += " - " + strconv.Itoa(scenarios) + " scenarios"
	}
	return name
}

// FCAFStepResult is how a main step of a pipeline run ended: Completed, or
// failed with the failure's Code and the Maestro output (Stdout) of a mobile
// step. A step that did not run has neither.
type FCAFStepResult struct {
	Completed bool
	Failed    bool
	Code      string
	Stdout    string
}

// FCAFStepOutcome is how one wallet action of a pipeline step did: the
// role it runs, the step, the scenario fixture the step belongs to
// (<scenario file>#<scenario step id>) and its status.
type FCAFStepOutcome struct {
	StepID        string `json:"step_id"`
	Action        string `json:"action"`
	Role          string `json:"role"`
	FixtureID     string `json:"fixture_id"`
	Status        string `json:"status"`
	FailedCommand string `json:"failed_command,omitempty"`
}

// FCAFRoleOutcome sums a role's step outcomes: failing when any step is
// failing, else ok when any is ok, else not_run. FailedCommand, FixtureID and
// StepID are those of its first failing step.
type FCAFRoleOutcome struct {
	Role          string `json:"role"`
	Status        string `json:"status"`
	OK            int    `json:"ok"`
	Failing       int    `json:"failing"`
	NotRun        int    `json:"not_run"`
	FailedCommand string `json:"failed_command,omitempty"`
	FixtureID     string `json:"fixture_id,omitempty"`
	StepID        string `json:"step_id,omitempty"`
}

// FCAFRoleOutcomes is the role_outcomes of a stored FCAF report: how each
// action of the pipeline's wallet did in the run, per step and per role.
type FCAFRoleOutcomes struct {
	// WalletPrefix is "<org>/<wallet>/" of the wallet the pipeline runs.
	WalletPrefix string            `json:"wallet_prefix"`
	Roles        []FCAFRoleOutcome `json:"roles"`
	Steps        []FCAFStepOutcome `json:"steps"`
}

// maestroLabelFailed is the line Maestro prints when the `runFlow` block of
// an action of a composed flow failed (pipeline.ActionFlowLabelPrefix).
var maestroLabelFailed = regexp.MustCompile(
	`^\s*` + regexp.QuoteMeta(pipeline.ActionFlowLabelPrefix) + `(.+?)\.\.\. FAILED\s*$`,
)

// FCAFRoleOutcomesOf reads, from a pipeline definition and how its steps
// ended, how every action of the pipeline's wallet did. The wallet is the
// one of the first mobile-automation step's first action (the prelude's
// reset); its actions are those under the same "<org>/<wallet>/" prefix,
// each named after its role. scenarioFiles maps a scenario's step ID prefix
// (aggregate.ScenarioPrefix) to its file name. It returns nil when no step
// runs a wallet action.
func FCAFRoleOutcomesOf(
	definition *pipeline.WorkflowDefinition,
	results map[string]FCAFStepResult,
	scenarioFiles map[string]string,
) *FCAFRoleOutcomes {
	if definition == nil {
		return nil
	}
	prefix := ""
	for _, step := range definition.Steps {
		if ids := stepActionIDs(step); len(ids) > 0 {
			prefix = path.Dir(ids[0]) + "/"
			break
		}
	}
	if prefix == "" {
		return nil
	}
	outcomes := &FCAFRoleOutcomes{
		WalletPrefix: prefix,
		Roles:        []FCAFRoleOutcome{},
		Steps:        []FCAFStepOutcome{},
	}
	for _, step := range definition.Steps {
		ids := stepActionIDs(step)
		statuses, failedCommand := actionStatuses(ids, results[step.ID])
		fixture := fixtureID(step.ID, scenarioFiles)
		for i, id := range ids {
			if path.Dir(id)+"/" != prefix {
				continue
			}
			outcome := FCAFStepOutcome{
				StepID:    step.ID,
				Action:    id,
				Role:      path.Base(id),
				FixtureID: fixture,
				Status:    statuses[i],
			}
			if outcome.Status == FCAFRunFailing {
				outcome.FailedCommand = failedCommand
			}
			outcomes.Steps = append(outcomes.Steps, outcome)
		}
	}
	for _, step := range outcomes.Steps {
		index := slices.IndexFunc(outcomes.Roles, func(r FCAFRoleOutcome) bool {
			return r.Role == step.Role
		})
		if index < 0 {
			outcomes.Roles = append(outcomes.Roles, FCAFRoleOutcome{Role: step.Role})
			index = len(outcomes.Roles) - 1
		}
		role := &outcomes.Roles[index]
		switch step.Status {
		case FCAFRunFailing:
			if role.Failing == 0 {
				role.FailedCommand, role.FixtureID, role.StepID =
					step.FailedCommand, step.FixtureID, step.StepID
			}
			role.Failing++
		case FCAFRunOK:
			role.OK++
		default:
			role.NotRun++
		}
	}
	for i := range outcomes.Roles {
		role := &outcomes.Roles[i]
		switch {
		case role.Failing > 0:
			role.Status = FCAFRunFailing
		case role.OK > 0:
			role.Status = FCAFRunOK
		default:
			role.Status = FCAFRunNotRun
		}
	}
	slices.SortFunc(outcomes.Roles, func(a, b FCAFRoleOutcome) int {
		return strings.Compare(a.Role, b.Role)
	})
	return outcomes
}

// stepActionIDs lists the wallet actions a mobile-automation step runs, in
// order: its action_id or the id of each of its actions.
func stepActionIDs(step pipeline.StepDefinition) []string {
	if step.Use != StepUseMobileAutomation || step.With.Payload == nil {
		return nil
	}
	if id := strings.TrimSpace(stringValue(step.With.Payload[mobileActionIDKey])); id != "" {
		return []string{id}
	}
	entries, _ := step.With.Payload[mobileActionsKey].([]any)
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		fields, _ := entry.(map[string]any)
		ids = append(ids, strings.TrimSpace(stringValue(fields["id"])))
	}
	return ids
}

// actionStatuses returns the status of each of a step's actions and the
// Maestro command the failing one stopped at. A completed step ran every
// action; a Maestro flow failure (CRE315) of a single action is that
// action's, and in a composed flow it is the one whose label Maestro reports
// as FAILED: the actions before it ran, those after it did not. Any other
// end leaves every action not run.
func actionStatuses(ids []string, result FCAFStepResult) ([]string, string) {
	statuses := make([]string, len(ids))
	fill := func(status string) {
		for i := range statuses {
			statuses[i] = status
		}
	}
	switch {
	case result.Completed:
		fill(FCAFRunOK)
		return statuses, ""
	case !result.Failed || result.Code != errorcodes.MobileAutomationStepError:
		fill(FCAFRunNotRun)
		return statuses, ""
	}
	fill(FCAFRunNotRun)
	command := failedMaestroCommand(result.Stdout)
	if len(ids) == 1 {
		statuses[0] = FCAFRunFailing
		return statuses, command
	}
	failed := ""
	for _, line := range strings.Split(result.Stdout, "\n") {
		if match := maestroLabelFailed.FindStringSubmatch(
			strings.TrimRight(line, "\r"),
		); match != nil {
			failed = strings.TrimSpace(match[1])
			break
		}
	}
	index := slices.Index(ids, failed)
	if index < 0 {
		return statuses, ""
	}
	for i := range index {
		statuses[i] = FCAFRunOK
	}
	statuses[index] = FCAFRunFailing
	return statuses, command
}

// failedMaestroCommand is the first command Maestro reports as FAILED that
// is not the label of an action's block (MaestroFailure).
func failedMaestroCommand(stdout string) string {
	for _, line := range strings.Split(stdout, "\n") {
		match := maestroFailedLine.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if match == nil {
			continue
		}
		command := strings.TrimSpace(match[1])
		if !strings.HasPrefix(command, pipeline.ActionFlowLabelPrefix) {
			return command
		}
	}
	command, _ := MaestroFailure(stdout)
	return command
}

// fixtureID is "<scenario file>#<scenario step id>" of an aggregate step ID
// that starts with a scenario's prefix, "" for another step (the prelude).
func fixtureID(stepID string, scenarioFiles map[string]string) string {
	for prefix, file := range scenarioFiles {
		if rest, ok := strings.CutPrefix(stepID, prefix+"-"); ok {
			return file + "#" + rest
		}
	}
	return ""
}

// FCAFRoleRun is how a role's action did in the wallet's last FCAF run.
type FCAFRoleRun struct {
	Status        string `json:"status"`
	FailedSteps   int    `json:"failed_steps"`
	FailedCommand string `json:"failed_command"`
	FixtureID     string `json:"fixture_id"`
	RunURL        string `json:"run_url"`
}

// FCAFRunSummary is the test summary of the wallet's last finished FCAF run.
type FCAFRunSummary struct {
	Pass          int    `json:"pass"`
	Fail          int    `json:"fail"`
	Blocked       int    `json:"blocked"`
	Skipped       int    `json:"skipped"`
	Inconclusive  int    `json:"inconclusive"`
	NotApplicable int    `json:"not_applicable"`
	Error         int    `json:"error"`
	Total         int    `json:"total"`
	Pipeline      string `json:"pipeline"`
	Created       string `json:"created"`
}

// FCAFLastRun is the wallet's last finished FCAF run: its summary, link and
// the outcome of each role it ran.
type FCAFLastRun struct {
	Summary FCAFRunSummary
	RunURL  string
	Roles   map[string]FCAFRoleRun
}

// storedFCAFReport is the part of a stored fcaf_report coverage reads.
type storedFCAFReport struct {
	Summary      FCAFRunSummary    `json:"summary"`
	RoleOutcomes *FCAFRoleOutcomes `json:"role_outcomes"`
}

// ReadFCAFLastRun reads the newest pipeline result with a stored FCAF report
// of the wallet's FCAF pipelines (FCAFPipelineName, complete or subset)
// owned by the selection's organization; nil when there is none.
func ReadFCAFLastRun(app core.App, selection Selection) (*FCAFLastRun, error) {
	base := FCAFPipelineName(selection.Wallet.GetString("name"), 0)
	subset := regexp.MustCompile(`^` + regexp.QuoteMeta(base) + ` - [0-9]+ scenarios$`)
	records, err := app.FindAllRecords(
		"pipelines",
		dbx.HashExp{"owner": selection.Organization.Id},
		dbx.Like("name", base).Match(false, true),
	)
	if err != nil {
		return nil, fmt.Errorf("list FCAF pipelines: %w", err)
	}
	pipelines := map[string]*core.Record{}
	ids := []any{}
	for _, record := range records {
		name := record.GetString("name")
		if name == base || subset.MatchString(name) {
			pipelines[record.Id] = record
			ids = append(ids, record.Id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	results, err := app.FindAllRecords(
		"pipeline_results",
		dbx.In("pipeline", ids...),
		dbx.Not(dbx.HashExp{"fcaf_report": ""}),
	)
	if err != nil {
		return nil, fmt.Errorf("list FCAF pipeline results: %w", err)
	}
	var newest *core.Record
	for _, result := range results {
		if newest == nil || result.GetDateTime("created").After(newest.GetDateTime("created")) {
			newest = result
		}
	}
	if newest == nil {
		return nil, nil
	}
	report, err := readStoredFCAFReport(app, newest)
	if err != nil {
		return nil, err
	}
	run := &FCAFLastRun{
		Summary: report.Summary,
		RunURL: utils.JoinURL(
			app.Settings().Meta.AppURL,
			"my", "tests", "runs",
			newest.GetString("workflow_id"),
			newest.GetString("run_id"),
		),
		Roles: map[string]FCAFRoleRun{},
	}
	summary := &run.Summary
	summary.Total = summary.Pass + summary.Fail + summary.Blocked + summary.Skipped +
		summary.Inconclusive + summary.NotApplicable + summary.Error
	if owner := pipelines[newest.GetString("pipeline")]; owner != nil {
		summary.Pipeline = selection.Organization.GetString("canonified_name") + "/" +
			owner.GetString("canonified_name")
	}
	summary.Created = newest.GetDateTime("created").String()
	if report.RoleOutcomes != nil {
		for _, role := range report.RoleOutcomes.Roles {
			run.Roles[role.Role] = FCAFRoleRun{
				Status:        role.Status,
				FailedSteps:   role.Failing,
				FailedCommand: role.FailedCommand,
				FixtureID:     role.FixtureID,
				RunURL:        run.RunURL,
			}
		}
	}
	return run, nil
}

func readStoredFCAFReport(app core.App, record *core.Record) (storedFCAFReport, error) {
	var report storedFCAFReport
	fsys, err := app.NewFilesystem()
	if err != nil {
		return report, fmt.Errorf("open files: %w", err)
	}
	defer fsys.Close()
	reader, err := fsys.GetReader(record.BaseFilesPath() + "/" + record.GetString("fcaf_report"))
	if err != nil {
		return report, fmt.Errorf("open FCAF report: %w", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return report, fmt.Errorf("read FCAF report: %w", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return report, fmt.Errorf("decode FCAF report: %w", err)
	}
	return report, nil
}

// FCAFWrittenRoles are the roles whose action is written and up to date
// (current or incomplete); a stale, missing or unmanaged role is missing.
func FCAFWrittenRoles(coverage []FCAFRoleCoverage) map[string]bool {
	written := make(map[string]bool, len(coverage))
	for _, role := range coverage {
		written[role.Role] = role.Status == FCAFCurrent || role.Status == FCAFIncomplete
	}
	return written
}
