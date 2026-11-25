// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/aggregate"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
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
// relying-party validation of the wallet walletPath (FCAFWalletPath): one
// per wallet, and for a subset (scenarios, its scenario file names, not
// empty) one per set of scenarios, suffixed " - scenarios <hash>" (the first
// 8 hex digits of the sha256 of the sorted names joined by newlines), so a
// subset never replaces the wallet's complete pipeline nor another subset.
func FCAFPipelineName(walletPath string, scenarios []string) string {
	name := fcafPipelineNamePrefix + walletPath
	if len(scenarios) > 0 {
		sorted := slices.Sorted(slices.Values(scenarios))
		sum := sha256.Sum256([]byte(strings.Join(sorted, "\n")))
		name += fcafSubsetInfix + hex.EncodeToString(sum[:])[:8]
	}
	return name
}

// fcafSubsetInfix follows the wallet path in a subset pipeline's name.
const fcafSubsetInfix = " - scenarios "

// FCAFWalletPath is "<owner>/<wallet>" of the selection's wallet, canonified.
func FCAFWalletPath(selection Selection) string {
	return selection.WalletOwner.GetString("canonified_name") + "/" +
		selection.Wallet.GetString("canonified_name")
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

// FCAFRoleOutcome sums the step outcomes of one wallet action: failing when
// any step is failing, else ok when any is ok, else not_run. Action is the
// action's path, which coverage matches against the wallet's actions; Role
// is its FCAF role (the saved fcaf_role, else the path's base name).
// FailedCommand, FixtureID and StepID are those of its first failing step.
type FCAFRoleOutcome struct {
	Action        string `json:"action"`
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
// reset); its actions are those under the same "<org>/<wallet>/" prefix.
// roleOf names an action's FCAF role ("" or a nil roleOf: the path's base
// name). scenarioFiles maps a scenario's step ID prefix
// (aggregate.ScenarioPrefix) to its file name. It returns nil when no step
// runs a wallet action.
func FCAFRoleOutcomesOf(
	definition *pipeline.WorkflowDefinition,
	results map[string]FCAFStepResult,
	scenarioFiles map[string]string,
	roleOf func(action string) string,
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
	roles := map[string]string{}
	role := func(action string) string {
		name, known := roles[action]
		if !known {
			if roleOf != nil {
				name = roleOf(action)
			}
			if name == "" {
				name = path.Base(action)
			}
			roles[action] = name
		}
		return name
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
				Role:      role(id),
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
			return r.Action == step.Action
		})
		if index < 0 {
			outcomes.Roles = append(outcomes.Roles, FCAFRoleOutcome{
				Action: step.Action,
				Role:   step.Role,
			})
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
		return cmp.Or(strings.Compare(a.Role, b.Role), strings.Compare(a.Action, b.Action))
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
// Status is FCAFRunState.Status of a run that ended without a report
// (completed, failed, canceled, terminated, timed_out or continued_as_new),
// empty otherwise.
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
	Status        string `json:"status"`
}

// FCAFLastRun is the wallet's last finished FCAF run: its summary, link and
// the outcome of each wallet action it ran, keyed by the action's path.
// NotRun, set when the run ended without a report, is the outcome of every
// action Actions does not name.
type FCAFLastRun struct {
	Summary FCAFRunSummary
	RunURL  string
	Actions map[string]FCAFRoleRun
	NotRun  *FCAFRoleRun
}

// Run is how the wallet action at path did in the last run; nil when the
// run did not run it.
func (l *FCAFLastRun) Run(path string) *FCAFRoleRun {
	if l == nil || path == "" {
		return nil
	}
	if run, ran := l.Actions[path]; ran {
		return &run
	}
	if l.NotRun != nil {
		run := *l.NotRun
		return &run
	}
	return nil
}

// FCAFRunState is how a pipeline run stands in Temporal: Finished once it
// closed, its Status, the lowercase Temporal status (running, completed,
// failed, canceled, terminated, timed_out or continued_as_new) and, for a
// failed run, the step failures of its close failure by step ID.
type FCAFRunState struct {
	Finished bool
	Status   string
	Failures map[string]StepFailure
}

// FCAFRunStateFunc reads the state of the pipeline run workflowID/runID.
type FCAFRunStateFunc func(workflowID, runID string) (FCAFRunState, error)

// storedFCAFReport is the part of a stored fcaf_report coverage reads.
type storedFCAFReport struct {
	Summary      FCAFRunSummary    `json:"summary"`
	RoleOutcomes *FCAFRoleOutcomes `json:"role_outcomes"`
}

// fcafFinishedRun is a finished run of one of the wallet's FCAF pipelines:
// its report, or its Temporal state when it ended without one.
type fcafFinishedRun struct {
	result   *core.Record
	pipeline *core.Record
	subset   bool
	report   *storedFCAFReport
	state    FCAFRunState
	url      string
}

// ReadFCAFLastRun reads the wallet's last FCAF run from the pipeline results
// of the selection's organization for the wallet's FCAF pipeline records
// (FCAFPipelineName of FCAFWalletPath, complete or subset). A result with a
// stored report finished; one without asks state, and counts only once
// finished (a nil state skips it). A report of another wallet's actions
// (role_outcomes.wallet_prefix) is ignored. The outcomes start from the
// newest finished complete run, else from the oldest finished subset run,
// and every newer finished subset run, oldest first, replaces the outcome of
// each action it ran. The summary and link are those of the newest finished
// complete run, else of the newest finished subset run. A run without a
// report leaves every action not_run when it is the starting run, except
// resetAction (the path of the wallet's reset action), failing when the
// prelude step failed. It returns nil when there is no finished run.
func ReadFCAFLastRun(
	app core.App,
	selection Selection,
	resetAction string,
	state FCAFRunStateFunc,
) (*FCAFLastRun, error) {
	walletPath := FCAFWalletPath(selection)
	base := FCAFPipelineName(walletPath, nil)
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
		if name == base || strings.HasPrefix(name, base+fcafSubsetInfix) {
			pipelines[record.Id] = record
			ids = append(ids, record.Id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	// Only the organization's own runs: another organization may run its own
	// YAML on this record.
	results, err := app.FindAllRecords(
		"pipeline_results",
		dbx.In("pipeline", ids...),
		dbx.HashExp{"owner": selection.Organization.Id},
	)
	if err != nil {
		return nil, fmt.Errorf("list FCAF pipeline results: %w", err)
	}
	slices.SortFunc(results, func(a, b *core.Record) int {
		return b.GetDateTime("created").Compare(a.GetDateTime("created"))
	})
	// The finished runs, newest first, down to the newest complete run.
	var runs []fcafFinishedRun
	for _, result := range results {
		run, finished, err := readFCAFFinishedRun(app, result, walletPath, state)
		if err != nil {
			return nil, err
		}
		if !finished {
			continue
		}
		run.pipeline = pipelines[result.GetString("pipeline")]
		run.subset = run.pipeline.GetString("name") != base
		runs = append(runs, run)
		if !run.subset {
			break
		}
	}
	if len(runs) == 0 {
		return nil, nil
	}
	// The newest complete run, else the newest subset run, gives the summary.
	shown := runs[len(runs)-1]
	if shown.subset {
		shown = runs[0]
	}
	last := &FCAFLastRun{RunURL: shown.url, Actions: map[string]FCAFRoleRun{}}
	if shown.report != nil {
		last.Summary = shown.report.Summary
	}
	summary := &last.Summary
	summary.Total = summary.Pass + summary.Fail + summary.Blocked + summary.Skipped +
		summary.Inconclusive + summary.NotApplicable + summary.Error
	summary.Pipeline = selection.Organization.GetString("canonified_name") + "/" +
		shown.pipeline.GetString("canonified_name")
	summary.Created = shown.result.GetDateTime("created").String()
	summary.Status = shown.state.Status
	// Oldest first, so the newest run of each action wins.
	for i := len(runs) - 1; i >= 0; i-- {
		runs[i].apply(last, resetAction, i == len(runs)-1)
	}
	return last, nil
}

// readFCAFFinishedRun reads a pipeline result as a finished run; finished
// is false while it runs, when state cannot tell, or when its report is of
// another wallet.
func readFCAFFinishedRun(
	app core.App,
	result *core.Record,
	walletPath string,
	state FCAFRunStateFunc,
) (fcafFinishedRun, bool, error) {
	run := fcafFinishedRun{
		result: result,
		url: pipeline.RunPageURL(
			app.Settings().Meta.AppURL,
			result.GetString("workflow_id"),
			result.GetString("run_id"),
		),
	}
	if result.GetString("fcaf_report") != "" {
		report, err := readStoredFCAFReport(app, result)
		if err != nil {
			return run, false, err
		}
		if report.RoleOutcomes != nil && report.RoleOutcomes.WalletPrefix != walletPath+"/" {
			return run, false, nil
		}
		run.report = &report
		return run, true, nil
	}
	if state == nil {
		return run, false, nil
	}
	current, err := state(result.GetString("workflow_id"), result.GetString("run_id"))
	if err != nil || !current.Finished {
		// Coverage stays available when Temporal is not.
		return run, false, nil //nolint:nilerr // an unknown state is not finished
	}
	run.state = current
	return run, true, nil
}

// apply sets the outcomes of the run's actions in last; with base, a run
// without a report also leaves every other action not run.
func (r fcafFinishedRun) apply(last *FCAFLastRun, resetAction string, base bool) {
	if r.report != nil {
		if r.report.RoleOutcomes == nil {
			return
		}
		for _, outcome := range r.report.RoleOutcomes.Roles {
			if outcome.Action == "" {
				continue
			}
			last.Actions[outcome.Action] = FCAFRoleRun{
				Status:        outcome.Status,
				FailedSteps:   outcome.Failing,
				FailedCommand: outcome.FailedCommand,
				FixtureID:     outcome.FixtureID,
				RunURL:        r.url,
			}
		}
		return
	}
	if base {
		last.NotRun = &FCAFRoleRun{Status: FCAFRunNotRun, RunURL: r.url}
	}
	if failure, failed := r.state.Failures[aggregate.PreludeStepID]; failed && resetAction != "" {
		failing := FCAFRoleRun{Status: FCAFRunFailing, FailedSteps: 1, RunURL: r.url}
		if failure.Code == errorcodes.MobileAutomationStepError {
			failing.FailedCommand = failedMaestroCommand(failure.Stdout)
		}
		last.Actions[resetAction] = failing
	}
}

// FCAFActionRoles names the FCAF role of a wallet action path: the saved
// fcaf_role of the action record, "" when the path resolves to none.
func FCAFActionRoles(app core.App) func(action string) string {
	return func(action string) string {
		record, err := canonify.Resolve(app, action)
		if err != nil || record == nil || record.Collection().Name != "wallet_actions" {
			return ""
		}
		return savedProvenance(record).FCAFRole
	}
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

// FCAFWrittenActions maps each role whose action is written and up to date
// (current or incomplete) to the action's path; a stale, missing or
// unmanaged role has none.
func FCAFWrittenActions(coverage []FCAFRoleCoverage) map[string]string {
	written := make(map[string]string, len(coverage))
	for _, role := range coverage {
		if role.Action != "" && (role.Status == FCAFCurrent || role.Status == FCAFIncomplete) {
			written[role.Role] = role.Action
		}
	}
	return written
}
