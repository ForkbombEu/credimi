// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// FCAFScope is the role contract, its templates root and the suite of a
// session that authors FCAF actions. Fix, set only for a fix session, maps
// each role it fixes to the contract fixtures the role's action failed on
// in the wallet's last FCAF run (FCAFFixCoverage, FCAFFixRoles).
type FCAFScope struct {
	Contract *walletroles.Contract
	Root     string // the FCAF templates root the contract was loaded from
	Suite    string // e.g. "wallet_solution/relying_party"
	Fix      map[string][]string
}

// Coverage statuses of an FCAF role in the session wallet. FCAFIncomplete is
// a current action whose replays miss a fixture of the role's current
// authoring set (walletroles.Contract.AuthoringSet).
const (
	FCAFMissing    = "missing"
	FCAFCurrent    = "current"
	FCAFIncomplete = "incomplete"
	FCAFStale      = "stale"
	FCAFUnmanaged  = "unmanaged"
)

// FCAFOutcome maps a contract outcome to the Spec outcome.
func FCAFOutcome(o walletroles.Outcome) string {
	switch o {
	case walletroles.OutcomeRejection:
		return OutcomeRejection
	case walletroles.OutcomeTolerant:
		return OutcomeTolerant
	case walletroles.OutcomeWait:
		return OutcomeWait
	default:
		return OutcomeSuccess
	}
}

// FCAFSuiteRole returns the role name of the scope's suite, ErrInvalid when
// the contract has no such role or the suite does not run it.
func FCAFSuiteRole(scope *FCAFScope, name string) (walletroles.Role, error) {
	role, ok := scope.Contract.Role(name)
	if !ok {
		return walletroles.Role{}, invalidf("fcaf_role: %s is not an FCAF role", name)
	}
	if !slices.Contains(role.Suites, scope.Suite) {
		return walletroles.Role{}, invalidf(
			"fcaf_role: %s is not part of the %s suite", role.Name, scope.Suite,
		)
	}
	return role, nil
}

// CheckFCAFPlan checks a normalized plan bound to an FCAF role against the
// contract: its slot, outcome, wait, parameters, starting actions and
// replaced action follow the role. Candidate prerequisites pass here; Save
// checks them once resolved.
func CheckFCAFPlan(scope *FCAFScope, spec *Spec) (walletroles.Role, error) {
	if scope == nil {
		return walletroles.Role{}, invalidf("fcaf_role: this session does not author FCAF actions")
	}
	role, err := FCAFSuiteRole(scope, spec.FCAFRole)
	if err != nil {
		return walletroles.Role{}, err
	}
	if slot, _ := SlotForCategory(role.Category); spec.Slot != slot {
		return walletroles.Role{}, invalidf(
			"fcaf_role: %s is a %s action: use the %s slot", role.Name, role.Category, slot,
		)
	}
	if outcome := FCAFOutcome(role.Outcome); spec.Outcome != outcome {
		return walletroles.Role{}, invalidf("fcaf_role: %s has the %s outcome", role.Name, outcome)
	}
	if spec.WaitSeconds != role.WaitSeconds {
		return walletroles.Role{}, invalidf("fcaf_role: %s waits %d s", role.Name, role.WaitSeconds)
	}
	given := ParameterNames(spec.Parameters)
	slices.Sort(given)
	wanted := slices.Sorted(slices.Values(role.Parameters))
	if !slices.Equal(given, wanted) {
		return walletroles.Role{}, invalidf(
			"fcaf_role: %s takes the parameters %v", role.Name, wanted,
		)
	}
	starts := scope.Contract.StartsFrom(role)
	startsMismatch := invalidf("fcaf_role: %s starts from %v, in that order", role.Name, starts)
	if len(spec.Prerequisites) != len(starts) {
		return walletroles.Role{}, startsMismatch
	}
	for i, prerequisite := range spec.Prerequisites {
		if prerequisite.Action != "" && path.Base(prerequisite.Action) != starts[i] {
			return walletroles.Role{}, startsMismatch
		}
	}
	if spec.Replaces != nil && path.Base(spec.Replaces.Action) != role.Name {
		return walletroles.Role{}, invalidf(
			"replaces: an FCAF %s action replaces only the wallet's %s action",
			role.Name, role.Name,
		)
	}
	return role, nil
}

// FCAFRoleCoverage is one role of the session suite and the state of the
// session wallet's action for it.
type FCAFRoleCoverage struct {
	Role       string   `json:"role"`
	Summary    string   `json:"summary"`
	Outcome    string   `json:"outcome"`
	Category   string   `json:"category"`
	StartsFrom []string `json:"starts_from"`
	// Status is "missing", "current", "incomplete", "stale" or "unmanaged".
	Status string `json:"status"`
	Action string `json:"action"` // canonified path of the wallet's record, "" when missing
	// Result is the agent's verdict on the saved FCAF action, "passed",
	// "unsupported" or "finding"; "" when not recorded or no action.
	Result     string `json:"result"`
	Note       string `json:"note"` // the agent's note on the saved action
	RoleSHA256 string `json:"role_sha256"`
	// MissingFixtures are the fixtures of the role's authoring set the saved
	// action was not replayed on, set when Status is incomplete; never nil.
	MissingFixtures []string `json:"missing_fixtures"`
	// Replays are the saved FCAF action's fcaf_replays, which a runner merges
	// with its replays of MissingFixtures; never nil.
	Replays []FCAFReplay `json:"fcaf_replays"`
	// Run is how the role's action did in the wallet's last finished FCAF
	// pipeline run (FCAFCoverageWithRun); nil when that run did not run it,
	// there is none or runs were not read. A failing run changes neither
	// Status nor Result.
	Run *FCAFRoleRun `json:"run"`
	// Verdict is the role's verdict: for a current or incomplete action, the
	// pipeline's (Run.Verdict: passed, finding, failing or blocked) when the
	// last run covered the role and the action was not saved after that run
	// (VerdictSource pipeline), else the agent's Result (VerdictSource
	// authoring: not verified yet); "" with an empty VerdictSource otherwise.
	Verdict       string `json:"verdict"`
	VerdictSource string `json:"verdict_source"`
	// Fix is set, only in a fix session's listing (FCAFFixCoverage), on the
	// roles the session fixes; FixFixtures are the contract fixtures their
	// action failed on in the wallet's last FCAF run, FailedCommand the
	// Maestro command it failed at.
	Fix           bool     `json:"fix,omitempty"`
	FixFixtures   []string `json:"fix_fixtures,omitempty"`
	FailedCommand string   `json:"failed_command,omitempty"`
}

// Where FCAFRoleCoverage.Verdict comes from.
const (
	FCAFVerdictFromPipeline  = "pipeline"
	FCAFVerdictFromAuthoring = "authoring"
)

// FCAFCoverage lists the roles of the session suite, the injected reset
// first, the injected issuance second, then in contract order, with the
// state of the caller organization's action of the same name for the
// selected wallet, without reading any run (every Run is nil; see
// FCAFCoverageWithRun). The second value is the FCAF profile of the newest
// FCAF action that saved one, nil when none did.
func FCAFCoverage(
	app core.App,
	selection Selection,
	scope *FCAFScope,
) ([]FCAFRoleCoverage, map[string]any, error) {
	coverage, _, profile, err := fcafCoverage(app, selection, scope)
	return coverage, profile, err
}

// fcafCoverage is FCAFCoverage with, per action path, when its record was
// last saved.
func fcafCoverage(
	app core.App,
	selection Selection,
	scope *FCAFScope,
) ([]FCAFRoleCoverage, map[string]types.DateTime, map[string]any, error) {
	records, err := app.FindAllRecords("wallet_actions", dbx.HashExp{
		"owner":  selection.Organization.Id,
		"wallet": selection.Wallet.Id,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("list wallet actions: %w", err)
	}
	updated := map[string]types.DateTime{}
	byName := make(map[string]*core.Record, len(records))
	var profile map[string]any
	var newest *core.Record
	for _, record := range records {
		byName[record.GetString("name")] = record
		provenance := savedProvenance(record)
		if provenance.FCAFRole == "" || provenance.FCAFProfile == nil {
			continue
		}
		if newest == nil ||
			record.GetDateTime("updated").After(newest.GetDateTime("updated")) {
			newest, profile = record, provenance.FCAFProfile
		}
	}
	coverage := []FCAFRoleCoverage{}
	for _, role := range fcafSuiteRoles(scope) {
		entry := FCAFRoleCoverage{
			Role:            role.Name,
			Summary:         role.Summary,
			Outcome:         FCAFOutcome(role.Outcome),
			Category:        role.Category,
			StartsFrom:      scope.Contract.StartsFrom(role),
			Status:          FCAFMissing,
			RoleSHA256:      role.SHA256(),
			MissingFixtures: []string{},
			Replays:         []FCAFReplay{},
		}
		if record, ok := byName[role.Name]; ok {
			entry.Action, err = walletActionPath(app, record)
			if err != nil {
				return nil, nil, nil, err
			}
			updated[entry.Action] = record.GetDateTime("updated")
			provenance := savedProvenance(record)
			switch {
			case provenance.FCAFRole != role.Name:
				entry.Status = FCAFUnmanaged
			case provenance.FCAFRoleSHA256 == entry.RoleSHA256:
				entry.Status = FCAFCurrent
				entry.MissingFixtures, err = fcafMissingFixtures(
					scope,
					role,
					provenance.FCAFReplays,
				)
				if err != nil {
					return nil, nil, nil, err
				}
				if len(entry.MissingFixtures) > 0 {
					entry.Status = FCAFIncomplete
				}
				if len(provenance.FCAFReplays) > 0 {
					entry.Replays = provenance.FCAFReplays
				}
			default:
				entry.Status = FCAFStale
			}
			if provenance.FCAFRole == role.Name && provenance.FCAFResult != nil {
				entry.Result = provenance.FCAFResult.Status
				entry.Note = provenance.FCAFResult.Note
			}
			if entry.Result != "" &&
				(entry.Status == FCAFCurrent || entry.Status == FCAFIncomplete) {
				entry.Verdict, entry.VerdictSource = entry.Result, FCAFVerdictFromAuthoring
			}
		}
		coverage = append(coverage, entry)
	}
	return coverage, updated, profile, nil
}

// FCAFCoverageWithRun is FCAFCoverage with how each role's action did in the
// wallet's last finished FCAF run (ReadFCAFLastRun with state, which may be
// nil, judged on the scope's walletroles Contract.RoleTests), matched by the
// action's path whatever its role's name; a current or incomplete action the
// run covered takes the pipeline's verdict unless it was saved after that
// run (FCAFLastRun.RanAfter). The third value is the last run, nil when
// there is none.
func FCAFCoverageWithRun(
	app core.App,
	selection Selection,
	scope *FCAFScope,
	state FCAFRunStateFunc,
) ([]FCAFRoleCoverage, map[string]any, *FCAFLastRun, error) {
	coverage, _, profile, lastRun, err := fcafCoverageWithRun(app, selection, scope, state)
	return coverage, profile, lastRun, err
}

// fcafCoverageWithRun is FCAFCoverageWithRun with, per action path, when its
// record was last saved.
func fcafCoverageWithRun(
	app core.App,
	selection Selection,
	scope *FCAFScope,
	state FCAFRunStateFunc,
) ([]FCAFRoleCoverage, map[string]types.DateTime, map[string]any, *FCAFLastRun, error) {
	coverage, updated, profile, err := fcafCoverage(app, selection, scope)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	resetRole := scope.Contract.Injected(walletroles.InjectedReset).Name
	resetAction := ""
	for _, entry := range coverage {
		if entry.Role == resetRole {
			resetAction = entry.Action
		}
	}
	roleTests, err := scope.Contract.RoleTests(scope.Root, scope.Suite)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	lastRun, err := ReadFCAFLastRun(app, selection, resetAction, roleTests, state)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for i := range coverage {
		entry := &coverage[i]
		entry.Run = lastRun.Run(entry.Action)
		if entry.Run != nil && entry.Run.Verdict != FCAFRunNotRun &&
			(entry.Status == FCAFCurrent || entry.Status == FCAFIncomplete) &&
			lastRun.RanAfter(entry.Action, updated[entry.Action]) {
			entry.Verdict, entry.VerdictSource = entry.Run.Verdict, FCAFVerdictFromPipeline
		}
	}
	return coverage, updated, profile, lastRun, nil
}

// FCAFFixCoverage is FCAFCoverageWithRun for a fix session that started at
// since: every role, those the session fixes marked Fix, with the fixtures
// their action failed on in the wallet's last FCAF run (Run.FailedFixtures)
// as FixFixtures and the Maestro command it failed at. A role is fixed when
// its action is current or incomplete, the run's verdict on it is failing,
// and coverage takes that verdict (the action was not saved since the run)
// or the action was saved at or after since, by the session itself, so a
// role stays fixed after the session saves it. With since now, the fixed
// roles are exactly those whose coverage verdict is failing.
func FCAFFixCoverage(
	app core.App,
	selection Selection,
	scope *FCAFScope,
	state FCAFRunStateFunc,
	since time.Time,
) ([]FCAFRoleCoverage, map[string]any, error) {
	coverage, updated, profile, _, err := fcafCoverageWithRun(app, selection, scope, state)
	if err != nil {
		return nil, nil, err
	}
	for i := range coverage {
		entry := &coverage[i]
		if entry.Run == nil || entry.Run.Verdict != FCAFRunFailing ||
			(entry.Status != FCAFCurrent && entry.Status != FCAFIncomplete) {
			continue
		}
		if entry.VerdictSource != FCAFVerdictFromPipeline &&
			updated[entry.Action].Time().Before(since) {
			continue
		}
		entry.Fix = true
		entry.FixFixtures = slices.Clone(entry.Run.FailedFixtures)
		entry.FailedCommand = entry.Run.FailedCommand
	}
	return coverage, profile, nil
}

// FCAFFixRoles returns, by role, the FixFixtures of the roles of coverage
// (FCAFFixCoverage) marked Fix, each never nil; empty when none is.
func FCAFFixRoles(coverage []FCAFRoleCoverage) map[string][]string {
	fix := map[string][]string{}
	for _, entry := range coverage {
		if entry.Fix {
			fix[entry.Role] = append([]string{}, entry.FixFixtures...)
		}
	}
	return fix
}

// FCAFFixFixtures returns the fixtures of the role, in Fixtures order, that
// a fix session's scope says its action failed on (FCAFScope.Fix); a failed
// step of a fixture the role does not have (an injected issuance or reset
// runs in another role's scenario step) has none. Empty outside a fix
// session.
func FCAFFixFixtures(scope *FCAFScope, role string) ([]walletroles.Fixture, error) {
	failed := scope.Fix[role]
	if len(failed) == 0 {
		return nil, nil
	}
	fixtures, err := scope.Contract.Fixtures(scope.Root, scope.Suite, role)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(fixtures, func(f walletroles.Fixture) bool {
		return !slices.Contains(failed, f.ID)
	}), nil
}

// FCAFKeyAttestationFinding is the FCAF finding about a wallet whose profile
// says pid_issuer jwt-proof: the agent wrote its actions with the jwt-proof
// PID only as an authoring aid, and FCAF pipelines still issue the
// key-attestation PID (the agent's fcaf.FCAF_KEY_ATTESTATION_FINDING).
const FCAFKeyAttestationFinding = "FCAF requirement not met: the wallet refuses Capture's " +
	"key-attestation PID (eu-pid-device-bound), which FCAF requires. The other actions were " +
	"written with a jwt-proof PID (eu-pid-jwt-proof-only) only to author and replay them; " +
	"FCAF pipelines still issue the key-attestation PID, so this wallet's conformance runs " +
	"will fail or block on it."

// FCAFWarnings returns the FCAF findings about the wallet of an FCAF
// profile: FCAFKeyAttestationFinding when its pid_issuer is jwt-proof. It is
// empty, never nil.
func FCAFWarnings(profile map[string]any) []string {
	if issuer, _ := profile["pid_issuer"].(string); issuer == walletroles.PIDIssuerJWTProof {
		return []string{FCAFKeyAttestationFinding}
	}
	return []string{}
}

// FCAFReplay is how an FCAF action did on one fixture of its role's
// authoring set: passed, finding (the action did the role and the fixture's
// FCAF tests report wallet findings) or action_failed.
type FCAFReplay struct {
	Fixture string `json:"fixture"`
	Class   string `json:"class"`
	Status  string `json:"status"`
}

// FCAF replay statuses, and MaxFCAFReplays the bound of an action's replays.
const (
	FCAFReplayPassed       = "passed"
	FCAFReplayFinding      = "finding"
	FCAFReplayActionFailed = "action_failed"
	MaxFCAFReplays         = 64
)

// CheckFCAFReplays refuses replays of an action without an FCAF role, more
// than 64 of them, and a replay of a fixture that is neither in the role's
// current authoring set nor, in a fix session, a fixture of the role it
// failed on (FCAFScope.Fix), given twice, of another class than the
// fixture's or with a status other than passed, finding or action_failed.
func CheckFCAFReplays(scope *FCAFScope, replays []FCAFReplay, fcafRole string) error {
	if len(replays) == 0 {
		return nil
	}
	if fcafRole == "" || scope == nil {
		return fmt.Errorf("%w: fcaf_replays: only FCAF actions carry replays", ErrInvalid)
	}
	if len(replays) > MaxFCAFReplays {
		return fmt.Errorf("%w: fcaf_replays: at most %d replays", ErrInvalid, MaxFCAFReplays)
	}
	set, err := scope.Contract.AuthoringSet(scope.Root, scope.Suite, fcafRole)
	if err != nil {
		return fmt.Errorf("fcaf_replays: %w", err)
	}
	fix, err := FCAFFixFixtures(scope, fcafRole)
	if err != nil {
		return fmt.Errorf("fcaf_replays: %w", err)
	}
	classes := make(map[string]string, len(set)+len(fix))
	for _, fixture := range slices.Concat(set, fix) {
		classes[fixture.ID] = fixture.Class
	}
	seen := map[string]bool{}
	for _, replay := range replays {
		class, ok := classes[replay.Fixture]
		switch {
		case !ok:
			return fmt.Errorf(
				"%w: fcaf_replays: %s is not a fixture of the authoring set of %s",
				ErrInvalid, replay.Fixture, fcafRole,
			)
		case seen[replay.Fixture]:
			return fmt.Errorf("%w: fcaf_replays: %s is replayed twice", ErrInvalid, replay.Fixture)
		case replay.Class != class:
			return fmt.Errorf(
				"%w: fcaf_replays: the class of %s is %q", ErrInvalid, replay.Fixture, class,
			)
		case replay.Status != FCAFReplayPassed && replay.Status != FCAFReplayFinding &&
			replay.Status != FCAFReplayActionFailed:
			return fmt.Errorf(
				"%w: fcaf_replays: status must be passed, finding or action_failed", ErrInvalid,
			)
		}
		seen[replay.Fixture] = true
	}
	return nil
}

// fcafMissingFixtures returns the fixtures of the role's authoring set that
// no replay covers, in authoring set order, never nil. The injected PID
// issuance needs none: its fixtures are alternative PID issuers.
func fcafMissingFixtures(
	scope *FCAFScope,
	role walletroles.Role,
	replays []FCAFReplay,
) ([]string, error) {
	missing := []string{}
	if role.Injected != "" || len(role.Parameters) == 0 {
		return missing, nil
	}
	set, err := scope.Contract.AuthoringSet(scope.Root, scope.Suite, role.Name)
	if err != nil {
		return nil, err
	}
	for _, fixture := range set {
		if !slices.ContainsFunc(
			replays,
			func(r FCAFReplay) bool { return r.Fixture == fixture.ID },
		) {
			missing = append(missing, fixture.ID)
		}
	}
	return missing, nil
}

// fcafSuiteRoles returns the roles of the scope suite: the injected reset,
// the injected issuance, then the others in contract order.
func fcafSuiteRoles(scope *FCAFScope) []walletroles.Role {
	roles := make([]walletroles.Role, 0, len(scope.Contract.Roles))
	for _, kind := range []walletroles.Injection{
		walletroles.InjectedReset,
		walletroles.InjectedIssuance,
	} {
		if role := scope.Contract.Injected(kind); slices.Contains(role.Suites, scope.Suite) {
			roles = append(roles, role)
		}
	}
	for _, role := range scope.Contract.Roles {
		if role.Injected == "" && slices.Contains(role.Suites, scope.Suite) {
			roles = append(roles, role)
		}
	}
	return roles
}

// fcafSuiteTags are the wallet_actions tags of the FCAF suites, in contract
// suite order.
var fcafSuiteTags = []struct{ suite, tag string }{
	{"wallet_solution/relying_party", "ws_rp"},
	{"wallet_solution/attestation_provider", "ws_ap"},
}

// fcafTags returns the tags an action of the role is saved with: fcaf, then
// the tag of each of the role's suites.
func fcafTags(role walletroles.Role) []string {
	tags := []string{"fcaf"}
	for _, suite := range fcafSuiteTags {
		if slices.Contains(role.Suites, suite.suite) {
			tags = append(tags, suite.tag)
		}
	}
	return tags
}

// mergeTags appends the tags missing from the comma-separated list tags,
// keeping its items, trimmed and without duplicates, and joins them with ", ".
func mergeTags(tags string, add []string) string {
	merged := []string{}
	for _, tag := range append(strings.Split(tags, ","), add...) {
		tag = strings.TrimSpace(tag)
		if tag != "" && !slices.Contains(merged, tag) {
			merged = append(merged, tag)
		}
	}
	return strings.Join(merged, ", ")
}
