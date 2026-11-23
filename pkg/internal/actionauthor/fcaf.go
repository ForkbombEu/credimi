// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"fmt"
	"path"
	"slices"

	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// FCAFScope is the role contract and suite of a session that authors FCAF
// actions.
type FCAFScope struct {
	Contract *walletroles.Contract
	Suite    string // e.g. "wallet_solution/relying_party"
}

// Coverage statuses of an FCAF role in the session wallet.
const (
	FCAFMissing   = "missing"
	FCAFCurrent   = "current"
	FCAFStale     = "stale"
	FCAFUnmanaged = "unmanaged"
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

// CheckFCAFPlan checks a normalized plan bound to an FCAF role against the
// contract: its slot, outcome, wait, parameters, starting actions and
// replaced action follow the role. Candidate prerequisites pass here; Save
// checks them once resolved.
func CheckFCAFPlan(scope *FCAFScope, spec *Spec) (walletroles.Role, error) {
	if scope == nil {
		return walletroles.Role{}, invalidf("fcaf_role: this session does not author FCAF actions")
	}
	role, ok := scope.Contract.Role(spec.FCAFRole)
	if !ok {
		return walletroles.Role{}, invalidf("fcaf_role: %s is not an FCAF role", spec.FCAFRole)
	}
	if !slices.Contains(role.Suites, scope.Suite) {
		return walletroles.Role{}, invalidf(
			"fcaf_role: %s is not part of the %s suite", role.Name, scope.Suite,
		)
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
	Status     string   `json:"status"` // "missing" | "current" | "stale" | "unmanaged"
	Action     string   `json:"action"` // canonified path of the wallet's record, "" when missing
	RoleSHA256 string   `json:"role_sha256"`
}

// FCAFCoverage lists the roles of the session suite, the injected reset
// first, the injected issuance second, then in contract order, with the
// state of the caller organization's action of the same name for the
// selected wallet. The second value is the FCAF profile of the newest FCAF
// action that saved one, nil when none did.
func FCAFCoverage(
	app core.App,
	selection Selection,
	scope *FCAFScope,
) ([]FCAFRoleCoverage, map[string]any, error) {
	records, err := app.FindAllRecords("wallet_actions", dbx.HashExp{
		"owner":  selection.Organization.Id,
		"wallet": selection.Wallet.Id,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("list wallet actions: %w", err)
	}
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
			Role:       role.Name,
			Summary:    role.Summary,
			Outcome:    FCAFOutcome(role.Outcome),
			Category:   role.Category,
			StartsFrom: scope.Contract.StartsFrom(role),
			Status:     FCAFMissing,
			RoleSHA256: role.SHA256(),
		}
		if record, ok := byName[role.Name]; ok {
			entry.Action, err = walletActionPath(app, record)
			if err != nil {
				return nil, nil, err
			}
			provenance := savedProvenance(record)
			switch {
			case provenance.FCAFRole != role.Name:
				entry.Status = FCAFUnmanaged
			case provenance.FCAFRoleSHA256 == entry.RoleSHA256:
				entry.Status = FCAFCurrent
			default:
				entry.Status = FCAFStale
			}
		}
		coverage = append(coverage, entry)
	}
	return coverage, profile, nil
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

// fcafSuiteTags are the wallet_actions tags of the FCAF suites.
var fcafSuiteTags = map[string]string{
	"wallet_solution/relying_party":        "ws_rp",
	"wallet_solution/attestation_provider": "ws_ap",
}

// suiteTag returns the short tag of an FCAF suite.
func suiteTag(suite string) string {
	return fcafSuiteTags[suite]
}

// fcafTags returns the tags FCAF actions of the suite are saved with.
func fcafTags(suite string) string {
	return "fcaf, " + suiteTag(suite)
}
