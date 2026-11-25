// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package aggregate builds the complete FCAF validation pipeline of a suite
// from its scenarios: every feasible scenario with prefixed step IDs, the
// wallet resets and PID issuances the wallet needs, and one final
// fcaf-validation step. cmd/fcaf-pipeline-gen writes the reference pipelines
// with it; Credimi builds a wallet's own pipeline with it.
package aggregate

import (
	"cmp"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"gopkg.in/yaml.v3"
)

const validationTask = "fcaf-validation"

type aggregateDefinition struct {
	Name    string           `yaml:"name"`
	Runtime map[string]any   `yaml:"runtime"`
	Steps   []map[string]any `yaml:"steps"`
}

var nonSlugCharacter = regexp.MustCompile(`[^a-z0-9]+`)

// The reference wallet consumes a credential instance per presentation and the
// Capture issuer advertises no batch issuance, so every presentation that can
// reach a successful share needs its own freshly issued PID. A presentation
// that the wallet refuses before consent leaves its instance untouched; it only
// needs the wallet to hold a PID of the requested format, which an earlier
// issuance of the same scenario or one injected issuance provides. An absence
// probe needs none: its request pins a value only a credential of its own
// scenario carries, so no injected PID can match it. How many instances each
// wallet action spends, whether it issues a credential (a credential offer the
// Wallet must refuse counts too) and whether it is an absence probe come from
// the role contract, fcaf-roles.yaml (walletRoles).
const mobileAutomationTask = "mobile-automation"

// credentialFormat separates the credential instances the wallet holds: an
// mdoc request cannot spend an SD-JWT VC instance, nor the other way round, and
// an SD-JWT VC request names its type through vct, so a degree request cannot
// spend a PID. formatSDJWT and formatMdoc are the PID formats, the only ones the
// generator issues; an SD-JWT VC of another type is keyed by its vct
// (sdJWTFormat). mdoc instances are keyed by format alone, because the domestic
// and the EU PID mdoc share the doctype eu.europa.ec.eudi.pid.1.
type credentialFormat string

const (
	formatSDJWT credentialFormat = "sd-jwt"
	formatMdoc  credentialFormat = "mdoc"
	// pidVCT is the vct of the PID SD-JWT VC; its Capture configurations start
	// with pidConfigurationPrefix instead.
	pidVCT                 = "urn:eudi:pid:1"
	pidConfigurationPrefix = "urn:eu.europa.ec.eudi:pid:1."
)

// sdJWTFormat keys an SD-JWT VC instance by its vct; the PID keeps formatSDJWT.
func sdJWTFormat(vct string) credentialFormat {
	if vct == "" || vct == pidVCT {
		return formatSDJWT
	}
	return credentialFormat(string(formatSDJWT) + " " + vct)
}

func isPIDFormat(format credentialFormat) bool {
	return format == formatSDJWT || format == formatMdoc
}

// resolveFormat maps a requested credential type the scenario never issued to
// the PID of its format: such a request is a near-miss the wallet must refuse,
// and the PID it holds is what makes the refusal mean more than an empty
// wallet. A type the scenario issued keeps its own key, so its instances are
// counted and spent on their own.
func resolveFormat(format credentialFormat, available map[credentialFormat]int) credentialFormat {
	if isPIDFormat(format) {
		return format
	}
	if _, issued := available[format]; issued {
		return format
	}
	return formatSDJWT
}

// deeplinkSource finds the step whose output a wallet step opens.
var deeplinkSource = regexp.MustCompile(`\$\{\{\s*([A-Za-z0-9_.-]+?)\.outputs\b`)

// SuiteSlug is the suite path as used in file names and scenario prefixes,
// e.g. wallet-solution-relying-party.
func SuiteSlug(suite string) string {
	return strings.NewReplacer("/", "-", "_", "-").Replace(suite)
}

// ScenarioPaths returns the scenario files of a suite under the FCAF
// templates root, sorted.
func ScenarioPaths(root string, suite string) ([]string, error) {
	dir := filepath.Join(root, suite, "scenarios")
	paths, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("find FCAF scenarios: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no FCAF scenarios found in %s", dir)
	}
	sort.Strings(paths)
	return paths, nil
}

// The reference pipelines' defaults: the device they run on and the
// reference wallet version their prelude installs.
const (
	DefaultDeviceID  = "forkbomb-bv-andrea/usb/device"
	DefaultVersionID = "forkbomb-bv-andrea/eudiw-beta-wallet/2026-09-42-demo"
	// PreludeStepID is the pipeline's first step, which installs, resets
	// and onboards the wallet.
	PreludeStepID = "onboard-reference-wallet"
	// ValidationStepID is the pipeline's final fcaf-validation step.
	ValidationStepID = "run-complete-fcaf-validation"
)

// ErrNothingToRun reports that the selected scenarios select no test, for
// example because the wallet lacks a role of every one of them.
var ErrNothingToRun = errors.New("FCAF scenarios select no tests")

// Options shape a built pipeline. The zero value builds the reference
// complete pipeline of Suite.
type Options struct {
	Suite string
	Name  string
	// ActionPrefix is "<org>/<wallet>/": every wallet action id under the
	// contract's ReferencePrefix moves under it. Empty keeps ReferencePrefix.
	ActionPrefix string
	// VersionID is the prelude's version_id; empty: DefaultVersionID.
	VersionID string
	// DeviceID is runtime.global_device_id; empty: DefaultDeviceID.
	DeviceID string
	// Roles are the roles the wallet has an action for; a scenario needing
	// another one is skipped whole. Nil runs every scenario.
	Roles map[string]bool
	// TestIDs, when not nil, keeps only these tests, each owned by a
	// selected scenario.
	TestIDs []string
	// FilterTestsByEvidence drops the tests whose exact evidence source no
	// selected scenario produces.
	FilterTestsByEvidence bool
	// NoVideo sets runtime.no_video when not nil; nil leaves the key out.
	NoVideo *bool
}

// SkippedScenario is a scenario left out because the wallet has no action
// for some of the roles its wallet steps run.
type SkippedScenario struct {
	Scenario     string   `json:"scenario"` // the scenario file name
	MissingRoles []string `json:"missing_roles"`
}

// Result is a built pipeline: its YAML, the scenarios left out and the tests
// its validation step runs.
type Result struct {
	YAML    []byte
	Skipped []SkippedScenario
	TestIDs []string
}

// Build builds the aggregate pipeline of the scenarios at paths (sorted, all
// of opts.Suite) under the FCAF templates root.
//
//nolint:gocyclo // One pass keeps the per-scenario wallet state in order.
func Build(root string, paths []string, opts Options) (Result, error) {
	if len(paths) == 0 {
		return Result{}, fmt.Errorf("no FCAF scenarios to build")
	}
	var allowedTests map[string]struct{}
	if opts.TestIDs != nil {
		allowedTests = make(map[string]struct{}, len(opts.TestIDs))
		for _, id := range opts.TestIDs {
			if strings.TrimSpace(id) == "" {
				return Result{}, fmt.Errorf("requested FCAF test ID is empty")
			}
			if _, exists := allowedTests[id]; exists {
				return Result{}, fmt.Errorf("requested FCAF test %q is duplicated", id)
			}
			allowedTests[id] = struct{}{}
		}
	}

	aggregate := aggregateDefinition{
		Name: opts.Name,
		Runtime: map[string]any{
			"global_device_id": cmp.Or(opts.DeviceID, DefaultDeviceID),
			"temporal": map[string]any{
				"activity_options": map[string]any{
					"schedule_to_close_timeout": "50m",
					"start_to_close_timeout":    "50m",
					"heartbeat_timeout":         "10m",
					"retry_policy": map[string]any{
						"maximum_attempts": 1,
					},
				},
			},
		},
		Steps: make([]map[string]any, 0, len(paths)*4),
	}
	if opts.NoVideo != nil {
		aggregate.Runtime["no_video"] = *opts.NoVideo
	}
	roles, err := loadWalletRoles(root)
	if err != nil {
		return Result{}, err
	}
	// The prelude installs and onboards the wallet with the reset role and
	// stops the run when it fails: without it no scenario runs.
	if reset := strings.TrimPrefix(roles.resetActionID, roles.prefix); opts.Roles != nil &&
		!opts.Roles[reset] {
		return Result{}, fmt.Errorf("%w: the wallet has no %s action", ErrNothingToRun, reset)
	}
	aggregate.Steps = append(
		aggregate.Steps,
		onboardingPrelude(roles, cmp.Or(opts.VersionID, DefaultVersionID)),
	)
	result := Result{}
	pipelineOutputs := map[string]any{}
	testIDs := map[string]struct{}{}
	seenStepIDs := map[string]string{}
	// Every scenario that drives the wallet starts from a reset wallet, so the
	// count of unspent instances starts again from zero there and an issuance
	// only covers presentations of its own scenario. The first one runs right
	// after onboardingPrelude, which already reset and onboarded the wallet.
	pidAvailable := map[credentialFormat]int{}
	walletUsed := false
	emitted := map[string]map[string]any{}

	for _, path := range paths {
		definition, err := walletroles.LoadScenario(path)
		if err != nil {
			return Result{}, err
		}
		prefix := ScenarioPrefix(opts.Suite, path)
		fixture := declaredFixture(definition.Runtime)
		stepIDs, err := scenarioStepIDs(path, definition.Steps, prefix)
		if err != nil {
			return Result{}, err
		}
		// The reset runs in the scenario's first wallet step when it can (see
		// injectWalletActions).
		var reset map[string]any
		usedBefore := walletUsed
		if drivesWallet(definition.Steps) {
			if walletUsed {
				reset = walletResetStep(prefix, roles)
				resetID, _ := reset["id"].(string)
				if err := claimStepID(seenStepIDs, resetID, path); err != nil {
					return Result{}, err
				}
			}
			walletUsed = true
			pidAvailable = map[credentialFormat]int{}
		}
		steps := []map[string]any{}
		validations := []map[string]any{}
		for _, step := range definition.Steps {
			use, _ := step["use"].(string)
			if use == validationTask {
				validations = append(validations, step)
				continue
			}

			rewritten := rewriteValue(step, stepIDs, fixture).(map[string]any)
			rewritten["continue_on_error"] = true
			id, _ := rewritten["id"].(string)
			if err := claimStepID(seenStepIDs, id, path); err != nil {
				return Result{}, err
			}
			var stepReset map[string]any
			if use == mobileAutomationTask {
				stepReset, reset = reset, nil
			}
			inserted, err := walletStepsBefore(
				rewritten,
				stepReset,
				emitted,
				pidAvailable,
				fixture,
				roles,
			)
			if err != nil {
				return Result{}, fmt.Errorf("FCAF scenario %s step %q: %w", path, id, err)
			}
			steps = append(steps, inserted...)
			steps = append(steps, rewritten)
			emitted[id] = rewritten
		}
		if missing := roles.missingRoles(steps, opts.Roles); len(missing) > 0 {
			// The next wallet scenario runs on the wallet as this one found it.
			walletUsed = usedBefore
			result.Skipped = append(result.Skipped, SkippedScenario{
				Scenario:     filepath.Base(path),
				MissingRoles: missing,
			})
			continue
		}
		for _, step := range validations {
			if err := mergeValidationStep(
				path,
				step,
				stepIDs,
				fixture,
				pipelineOutputs,
				testIDs,
				allowedTests,
			); err != nil {
				return Result{}, err
			}
		}
		linkScenarioBlocks(steps)
		aggregate.Steps = append(aggregate.Steps, steps...)
	}

	for _, id := range opts.TestIDs {
		if _, selected := testIDs[id]; !selected {
			return Result{}, fmt.Errorf(
				"requested FCAF test %q is not owned by selected scenarios",
				id,
			)
		}
	}
	if opts.FilterTestsByEvidence {
		testDefinitions, err := catalog.LoadTests(filepath.Join(root, opts.Suite, "tests"))
		if err != nil {
			return Result{}, fmt.Errorf("load FCAF test definitions: %w", err)
		}
		if err := removeTestsWithoutAvailableEvidence(
			testIDs,
			pipelineOutputs,
			testDefinitions,
		); err != nil {
			return Result{}, err
		}
	}

	selectedTests := make([]string, 0, len(testIDs))
	for id := range testIDs {
		selectedTests = append(selectedTests, id)
	}
	sort.Strings(selectedTests)
	if len(selectedTests) == 0 {
		return Result{}, ErrNothingToRun
	}

	aggregate.Steps = append(aggregate.Steps, map[string]any{
		"id":  ValidationStepID,
		"use": validationTask,
		"with": map[string]any{
			"test_ids":         selectedTests,
			"suite":            opts.Suite,
			"pipeline_outputs": pipelineOutputs,
		},
	})
	if opts.ActionPrefix != "" && opts.ActionPrefix != roles.prefix {
		for _, step := range aggregate.Steps {
			roles.movePrefix(step, opts.ActionPrefix)
		}
	}

	result.YAML, err = yaml.Marshal(aggregate)
	if err != nil {
		return Result{}, fmt.Errorf("encode aggregate FCAF pipeline: %w", err)
	}
	result.TestIDs = selectedTests
	return result, nil
}

// missingRoles returns, sorted, the roles of the wallet actions a
// scenario's steps run that are not in have, plus the reset role that
// prepares the wallet for any of them. A nil have misses nothing.
func (r walletRoles) missingRoles(steps []map[string]any, have map[string]bool) []string {
	if have == nil {
		return nil
	}
	missing := []string{}
	need := func(actionID string) {
		name, reference := strings.CutPrefix(actionID, r.prefix)
		if reference && !have[name] && !slices.Contains(missing, name) {
			missing = append(missing, name)
		}
	}
	drives := false
	for _, step := range steps {
		for _, id := range walletActionIDs(step) {
			drives = true
			need(id)
		}
	}
	if drives {
		need(r.resetActionID)
	}
	sort.Strings(missing)
	return missing
}

// walletActionIDs lists the action ids a mobile-automation step runs: its
// action_id or the id of each of its actions.
func walletActionIDs(step map[string]any) []string {
	if use, _ := step["use"].(string); use != mobileAutomationTask {
		return nil
	}
	with, _ := step["with"].(map[string]any)
	if id, _ := with["action_id"].(string); id != "" {
		return []string{id}
	}
	entries, _ := with["actions"].([]any)
	ids := make([]string, 0, len(entries))
	for _, raw := range entries {
		entry, _ := raw.(map[string]any)
		if id, _ := entry["id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// movePrefix moves a mobile-automation step's action ids from the
// contract's ReferencePrefix to prefix; nothing else of the step changes.
func (r walletRoles) movePrefix(step map[string]any, prefix string) {
	if use, _ := step["use"].(string); use != mobileAutomationTask {
		return
	}
	move := func(fields map[string]any) {
		if id, _ := fields["id"].(string); strings.HasPrefix(id, r.prefix) {
			fields["id"] = prefix + strings.TrimPrefix(id, r.prefix)
		}
	}
	with, _ := step["with"].(map[string]any)
	if id, _ := with["action_id"].(string); strings.HasPrefix(id, r.prefix) {
		with["action_id"] = prefix + strings.TrimPrefix(id, r.prefix)
	}
	entries, _ := with["actions"].([]any)
	for _, raw := range entries {
		if entry, ok := raw.(map[string]any); ok {
			move(entry)
		}
	}
}

// claimStepID records a generated step id, rejecting one an earlier scenario
// already produced.
func claimStepID(seen map[string]string, id string, path string) error {
	if previous, exists := seen[id]; exists {
		return fmt.Errorf("generated step id %q collides between %s and %s", id, previous, path)
	}
	seen[id] = path
	return nil
}

// pidIssuancesBefore updates the scenario's unspent PIDs for a step and returns
// the issuer sessions and the wallet actions accepting their offers that the
// step needs first (see injectIssuances). A presentation that shares spends one
// instance per Share, and a query with multiple: true spends every instance of
// its format the wallet holds, because the wallet returns all of them. A
// presentation the wallet refuses or cannot satisfy spends nothing, but its
// evidence only means something when the wallet holds a PID of the requested
// format: otherwise a rejection or a no-match screen is just the empty wallet.
// That PID stays unspent until the next reset. An absence probe gets none (see
// probesAbsence). A presentation of another credential type the scenario
// issued is an error when no instance of it is left, because the generator
// issues only PIDs and no PID can answer it.
func pidIssuancesBefore(
	step map[string]any,
	source map[string]any,
	consumed int,
	available map[credentialFormat]int,
	fixture map[string]string,
	roles walletRoles,
) ([]map[string]any, []map[string]any, error) {
	if issuesCredential(step, roles) {
		available[issuedCredentialFormat(source)]++
	}
	id, _ := step["id"].(string)
	var sessions []map[string]any
	var issuances []map[string]any
	var missing error
	issue := func(format credentialFormat) bool {
		if !isPIDFormat(format) {
			if missing == nil {
				missing = fmt.Errorf(
					"presentation %q needs an unspent %q instance and the generator issues "+
						"only PIDs: issue one in the scenario before it",
					id,
					format,
				)
			}
			return false
		}
		session, issuance := pidIssuance(
			id,
			len(issuances),
			format,
			fixture,
			roles.issuanceActionID,
		)
		sessions = append(sessions, session)
		issuances = append(issuances, issuance)
		return true
	}
	for _, format := range presentedCredentialFormats(source, consumed, available) {
		for need := consumed; need > 0; need-- {
			if available[format] > 0 {
				available[format]--
				continue
			}
			issue(format)
		}
	}
	if consumed > 0 {
		for _, format := range formatsSpentEntirely(source, available) {
			available[format] = 0
		}
	}
	if consumed > 0 || !opensPresentationRequest(step, source) || probesAbsence(step, roles) {
		return sessions, issuances, missing
	}
	for _, format := range presentedCredentialFormats(source, 1, available) {
		if available[format] > 0 {
			continue
		}
		if issue(format) {
			available[format]++
		}
	}
	return sessions, issuances, missing
}

// walletStepsBefore returns the steps to insert before a scenario step: the
// PIDs its presentation needs (pidIssuancesBefore) and its scenario's wallet
// reset, run inside the step's own flow when they can be (injectWalletActions).
func walletStepsBefore(
	step map[string]any,
	reset map[string]any,
	emitted map[string]map[string]any,
	available map[credentialFormat]int,
	fixture map[string]string,
	roles walletRoles,
) ([]map[string]any, error) {
	consumed, err := credentialInstancesConsumed(step, roles)
	if err != nil {
		return nil, err
	}
	sessions, issuances, err := pidIssuancesBefore(
		step,
		deeplinkSourceStep(step, emitted),
		consumed,
		available,
		fixture,
		roles,
	)
	if err != nil {
		return nil, err
	}
	return injectWalletActions(step, reset, sessions, issuances, roles)
}

// injectWalletActions returns the steps to insert before a wallet step for
// the wallet actions it needs first: the scenario's wallet reset when it is
// the scenario's first wallet step, then the PIDs issued for it. When every
// such action drives the step's own app, the step lists them, followed by its
// own action, as `actions` run in one Maestro flow, which saves a
// mobile-automation run each; a failed one then fails the step, which would
// otherwise run on a wallet still holding an earlier scenario's credentials or
// lacking the PID it needs. A step driving another app, such as the browser of
// a DC API flow, cannot run wallet commands, so its issuances stay separate
// steps, the first of which runs the reset when it can.
func injectWalletActions(
	step map[string]any,
	reset map[string]any,
	sessions []map[string]any,
	issuances []map[string]any,
	roles walletRoles,
) ([]map[string]any, error) {
	if reset == nil && len(issuances) == 0 {
		return sessions, nil
	}
	if use, _ := step["use"].(string); use != mobileAutomationTask {
		return nil, fmt.Errorf("a %q step cannot run wallet actions", use)
	}
	with, _ := step["with"].(map[string]any)
	if code, _ := with["action_code"].(string); code != "" {
		return nil, inlineActionCodeError()
	}
	actionID, _ := with["action_id"].(string)
	stepRole, err := roles.role(actionID)
	if err != nil {
		return nil, err
	}

	before := make([]any, 0, len(issuances)+2)
	if reset != nil {
		before = append(before, map[string]any{"id": roles.resetActionID})
	}
	combinable := true
	for _, issuance := range issuances {
		before = append(before, issuance)
	}
	for _, entry := range before {
		id, _ := entry.(map[string]any)["id"].(string)
		role, err := roles.role(id)
		if err != nil {
			return nil, err
		}
		combinable = combinable && role.HeaderApp == stepRole.HeaderApp
	}

	if combinable {
		own := map[string]any{"id": actionID}
		if parameters, ok := with["parameters"]; ok {
			own["parameters"] = parameters
		}
		delete(with, "action_id")
		delete(with, "parameters")
		with["actions"] = append(before, own)
		return sessions, nil
	}
	steps := make([]map[string]any, 0, 2*len(issuances)+1)
	for i, issuance := range issuances {
		sessionID, _ := sessions[i]["id"].(string)
		issue := map[string]any{
			"id":                strings.TrimSuffix(sessionID, "-session"),
			"use":               mobileAutomationTask,
			"continue_on_error": true,
			"with": map[string]any{
				"action_id":  issuance["id"],
				"version_id": "installed_from_external_source",
				"parameters": issuance["parameters"],
			},
		}
		resetSteps, err := injectWalletActions(issue, reset, nil, nil, roles)
		if err != nil {
			return nil, err
		}
		reset = nil
		steps = append(steps, sessions[i])
		steps = append(steps, resetSteps...)
		steps = append(steps, issue)
	}
	if reset != nil {
		steps = append(steps, reset)
	}
	return steps, nil
}

func removeTestsWithoutAvailableEvidence(
	testIDs map[string]struct{},
	pipelineOutputs map[string]any,
	testDefinitions map[string]dsl.TestDefinition,
) error {
	for id := range testIDs {
		test, exists := testDefinitions[id]
		if !exists {
			return fmt.Errorf("FCAF test %q has no definition", id)
		}
		for evidenceName, binding := range test.Evidence {
			source, outputName, found := strings.Cut(binding.From, ".outputs.")
			if !found || source == "" || outputName == "" {
				return fmt.Errorf(
					"FCAF test %q evidence %q has invalid source %q",
					id,
					evidenceName,
					binding.From,
				)
			}
			rawSource, exists := pipelineOutputs[source]
			if !exists {
				delete(testIDs, id)
				break
			}
			sourceDefinition, ok := rawSource.(map[string]any)
			if !ok {
				return fmt.Errorf("FCAF evidence source %q has invalid definition", source)
			}
			outputs, ok := sourceDefinition["output"].(map[string]any)
			if !ok {
				return fmt.Errorf("FCAF evidence source %q has no output object", source)
			}
			if _, exists := outputs[outputName]; !exists {
				delete(testIDs, id)
				break
			}
		}
	}
	return nil
}

// onboardingPrelude is the pipeline's first step. Its version_id makes the
// runner install a fresh Wallet, and it is the only Wallet step without
// continue_on_error, so a broken device stops the run before any scenario.
func onboardingPrelude(roles walletRoles, versionID string) map[string]any {
	return map[string]any{
		"id":  PreludeStepID,
		"use": mobileAutomationTask,
		"with": map[string]any{
			"action_id":  roles.resetActionID,
			"version_id": versionID,
		},
	}
}

// drivesWallet reports whether a scenario runs any step on the wallet, which is
// when it needs to start from a reset wallet.
func drivesWallet(steps []map[string]any) bool {
	for _, step := range steps {
		if use, _ := step["use"].(string); use == mobileAutomationTask {
			return true
		}
	}
	return false
}

// walletResetStep resets and onboards the wallet before a scenario, so its
// presentations cannot select a credential an earlier scenario issued and left
// unspent.
func walletResetStep(prefix string, roles walletRoles) map[string]any {
	return map[string]any{
		"id":                prefix + "-reset-wallet",
		"use":               mobileAutomationTask,
		"continue_on_error": true,
		"with": map[string]any{
			"action_id":  roles.resetActionID,
			"version_id": "installed_from_external_source",
		},
	}
}

// opensPresentationRequest reports whether a wallet step opens a verifier
// session created through the Capture OpenID4VP API.
func opensPresentationRequest(step map[string]any, source map[string]any) bool {
	if use, _ := step["use"].(string); use != mobileAutomationTask || source == nil {
		return false
	}
	with, _ := source["with"].(map[string]any)
	url, _ := with["url"].(string)
	return strings.HasSuffix(url, "/openid4vp/sessions")
}

// probesAbsence reports whether a wallet step is an absence probe: a request
// pinned to a value only a credential of its own scenario carries, which passes
// only when the wallet offers nothing for it. Issuing a PID before it adds no
// evidence, because no PID the generator issues can match the pinned value.
func probesAbsence(step map[string]any, roles walletRoles) bool {
	role, known := roles.stepRole(step)
	return known && role.AbsenceProbe
}

// walletRoles is the FCAF role contract keyed by the action id generated
// pipelines reference, the contract's ReferencePrefix + role name.
type walletRoles struct {
	byActionID       map[string]walletroles.Role
	prefix           string
	resetActionID    string
	issuanceActionID string
}

func loadWalletRoles(fcafRoot string) (walletRoles, error) {
	contract, err := walletroles.Load(fcafRoot)
	if err != nil {
		return walletRoles{}, err
	}
	prefix := contract.ReferencePrefix
	roles := walletRoles{
		byActionID:       make(map[string]walletroles.Role, len(contract.Roles)),
		prefix:           prefix,
		resetActionID:    prefix + contract.Injected(walletroles.InjectedReset).Name,
		issuanceActionID: prefix + contract.Injected(walletroles.InjectedIssuance).Name,
	}
	for _, role := range contract.Roles {
		roles.byActionID[prefix+role.Name] = role
	}
	return roles, nil
}

func (r walletRoles) role(actionID string) (walletroles.Role, error) {
	role, known := r.byActionID[actionID]
	if !known {
		return walletroles.Role{}, fmt.Errorf(
			"wallet action %q is not a role of %s",
			actionID,
			walletroles.FileName,
		)
	}
	return role, nil
}

// stepRole returns the role a mobile step's action_id runs, if any.
func (r walletRoles) stepRole(step map[string]any) (walletroles.Role, bool) {
	if use, _ := step["use"].(string); use != mobileAutomationTask {
		return walletroles.Role{}, false
	}
	with, _ := step["with"].(map[string]any)
	actionID, _ := with["action_id"].(string)
	role, known := r.byActionID[actionID]
	return role, known
}

// inlineActionCodeError refuses a wallet step whose flow is not a contract
// role: the generator can only reason about roles.
func inlineActionCodeError() error {
	return fmt.Errorf(
		"inline action_code is not supported: store the flow as a role in %s",
		walletroles.FileName,
	)
}

// issuesCredential reports whether a step adds, or may add, a credential
// instance to the wallet, which the next consuming presentation can spend. A
// credential offer the Wallet must refuse counts: the probe that follows it
// targets exactly that credential, so a Wallet that kept it spends it there,
// and a PID injected before the probe could never answer it.
func issuesCredential(step map[string]any, roles walletRoles) bool {
	role, known := roles.stepRole(step)
	return known && role.IssuesCredential
}

// credentialInstancesConsumed reports how many credential instances a step
// spends: one per Share its role performs.
func credentialInstancesConsumed(step map[string]any, roles walletRoles) (int, error) {
	if use, _ := step["use"].(string); use != mobileAutomationTask {
		return 0, nil
	}
	with, _ := step["with"].(map[string]any)
	if code, _ := with["action_code"].(string); code != "" {
		return 0, inlineActionCodeError()
	}
	actionID, _ := with["action_id"].(string)
	if actionID == "" {
		return 0, nil
	}
	role, err := roles.role(actionID)
	if err != nil {
		return 0, err
	}
	return role.Presents, nil
}

// deeplinkSourceStep returns the earlier step whose output a wallet step opens,
// or nil when no parameter references one.
func deeplinkSourceStep(step map[string]any, earlier map[string]map[string]any) map[string]any {
	with, _ := step["with"].(map[string]any)
	parameters, _ := with["parameters"].(map[string]any)
	names := make([]string, 0, len(parameters))
	for name := range parameters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		text, _ := parameters[name].(string)
		if match := deeplinkSource.FindStringSubmatch(text); match != nil {
			return earlier[match[1]]
		}
	}
	return nil
}

// deeplinkProducers are the step kinds whose output is a single-use deeplink.
var deeplinkProducers = map[string]struct{}{
	"credential-offer":               {},
	"use-case-verification-deeplink": {},
	"http-request":                   {},
}

// linkScenarioBlocks links every wallet step of one scenario to the deeplink
// producers it opens, together with the steps between them, so a retry runs
// the whole range again with a fresh deeplink. Overlapping ranges merge into
// one chain. Inside a chain every step runs once per attempt and the last
// step's retry policy sets the chain's attempts, so a policy set on an earlier
// step moves to the last one, unless that step has its own.
func linkScenarioBlocks(steps []map[string]any) {
	index := make(map[string]int, len(steps))
	for i, step := range steps {
		if id, _ := step["id"].(string); id != "" {
			index[id] = i
		}
	}
	for j, step := range steps {
		if use, _ := step["use"].(string); use != mobileAutomationTask {
			continue
		}
		with, err := yaml.Marshal(step["with"])
		if err != nil {
			continue
		}
		start := j
		for _, match := range deeplinkSource.FindAllStringSubmatch(string(with), -1) {
			i, found := index[match[1]]
			if !found || i >= j {
				continue
			}
			if use, _ := steps[i]["use"].(string); !isDeeplinkProducer(use) {
				continue
			}
			start = min(start, i)
		}
		for k := start + 1; k <= j; k++ {
			steps[k]["linked_to"] = steps[k-1]["id"]
		}
	}
	var carried any
	for k, step := range steps {
		options, _ := step["activity_options"].(map[string]any)
		if k+1 < len(steps) && steps[k+1]["linked_to"] != nil {
			if policy, ok := options["retry_policy"]; ok {
				if carried == nil {
					carried = policy
				}
				delete(options, "retry_policy")
				if len(options) == 0 {
					delete(step, "activity_options")
				}
			}
			continue
		}
		if carried != nil && step["linked_to"] != nil {
			if options == nil {
				options = map[string]any{}
				step["activity_options"] = options
			}
			if _, ok := options["retry_policy"]; !ok {
				options["retry_policy"] = carried
			}
		}
		carried = nil
	}
}

func isDeeplinkProducer(use string) bool {
	_, ok := deeplinkProducers[use]
	return ok
}

// issuedCredentialFormat reports the credential an issuance adds, read from
// the issuer session or credential offer that produced its deeplink. A Capture
// configuration names its type before `.sd-jwt` or `.mdoc`; a credential offer
// of a Credimi credential record is a PID.
func issuedCredentialFormat(source map[string]any) credentialFormat {
	with, _ := source["with"].(map[string]any)
	identifier, _ := with["credential_id"].(string)
	configuration := ""
	if body, ok := with["body"].(map[string]any); ok {
		configuration, _ = body["credential_configuration_id"].(string)
		if configuration != "" {
			identifier = configuration
		}
	}
	if strings.Contains(strings.ToLower(identifier), "mdoc") {
		return formatMdoc
	}
	if vct, _, found := strings.Cut(configuration, ".sd-jwt"); found &&
		!strings.HasPrefix(configuration, pidConfigurationPrefix) {
		return sdJWTFormat(vct)
	}
	return formatSDJWT
}

// presentedCredentialFormats lists the credentials a consuming presentation
// spends, read from the DCQL query of the verifier session it opens and
// resolved against what the scenario issued (resolveFormat). A query asking
// for both formats spends one instance of each per Share.
func presentedCredentialFormats(
	source map[string]any,
	consumed int,
	available map[credentialFormat]int,
) []credentialFormat {
	if consumed == 0 {
		return nil
	}
	formats := make([]credentialFormat, 0, 2)
	seen := map[credentialFormat]bool{}
	for _, credential := range sessionCredentialQueries(source) {
		format := resolveFormat(credentialQueryFormat(credential), available)
		if !seen[format] {
			seen[format] = true
			formats = append(formats, format)
		}
	}
	if len(formats) == 0 {
		return []credentialFormat{formatSDJWT}
	}
	return formats
}

// formatsSpentEntirely lists the credentials whose query in the session's DCQL
// query sets multiple: true. The wallet answers such a query with every
// matching instance it holds, so a Share spends all of them and a later
// presentation of the same scenario finds none left.
func formatsSpentEntirely(
	source map[string]any,
	available map[credentialFormat]int,
) []credentialFormat {
	var formats []credentialFormat
	for _, credential := range sessionCredentialQueries(source) {
		if multiple, _ := credential["multiple"].(bool); multiple {
			formats = append(formats, resolveFormat(credentialQueryFormat(credential), available))
		}
	}
	return formats
}

// sessionCredentialQueries returns the DCQL credential queries of the verifier
// session a presentation opens.
func sessionCredentialQueries(source map[string]any) []map[string]any {
	with, _ := source["with"].(map[string]any)
	body, _ := with["body"].(map[string]any)
	query := nestedMap(body, "dcql_query")
	if len(query) == 0 {
		query = nestedMap(nestedMap(body, "presentation_request"), "dcql_query")
	}
	items, _ := query["credentials"].([]any)
	credentials := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if credential, ok := item.(map[string]any); ok {
			credentials = append(credentials, credential)
		}
	}
	return credentials
}

// credentialQueryFormat reports the credential a DCQL credential query asks
// for: the mdoc PID format, the SD-JWT PID when its vct_values include the PID
// vct (or name none), and otherwise its first vct.
func credentialQueryFormat(credential map[string]any) credentialFormat {
	if declared, _ := credential["format"].(string); declared == "mso_mdoc" {
		return formatMdoc
	}
	meta, _ := credential["meta"].(map[string]any)
	values, _ := meta["vct_values"].([]any)
	first := ""
	for _, value := range values {
		vct, _ := value.(string)
		if vct == pidVCT {
			return formatSDJWT
		}
		if first == "" {
			first = vct
		}
	}
	return sdJWTFormat(first)
}

// pidIssuance issues one fresh PID of the given format for the presentation
// it precedes: an issuer session step with the contract's PID issuance
// request (walletroles.PIDIssuance) sent to the scenario's issuer, and the
// `actions` entry that accepts its offer, which runs in the presentation's
// own flow.
func pidIssuance(
	presentationID string,
	index int,
	format credentialFormat,
	fixture map[string]string,
	issuanceActionID string,
) (map[string]any, map[string]any) {
	base := "${fixture.verifier_url}"
	if _, declared := fixture["issuer_url"]; declared {
		base = "${fixture.issuer_url}"
	}
	// Inserted steps miss the scenario rewrite; an unresolved token would fall
	// back to the default issuer at run time instead of the scenario issuer.
	base = walletroles.SubstituteFixture(base, fixture)
	request := walletroles.PIDIssuance(walletroles.PIDIssuerKeyAttestation, string(format))
	// PIDIssuance's URL is a constant absolute URL; only its path is kept.
	requestURL, _ := url.Parse(request.URL)
	suffix := ""
	if index > 0 {
		suffix = fmt.Sprintf("-%d", index+1)
	}
	sessionID := presentationID + "-issue-pid" + suffix + "-session"
	session := map[string]any{
		"id":                sessionID,
		"use":               "http-request",
		"continue_on_error": true,
		"with": map[string]any{
			"method":          request.Method,
			"url":             base + requestURL.Path,
			"expected_status": 201,
			"headers":         request.Headers,
			"body":            request.Body,
		},
	}
	issuance := map[string]any{
		"id": issuanceActionID,
		"parameters": map[string]any{
			"deeplink": "${{ " + sessionID + ".outputs.body.deeplink }}",
		},
	}
	return session, issuance
}

func scenarioStepIDs(
	path string,
	steps []map[string]any,
	prefix string,
) (map[string]string, error) {
	ids := map[string]string{}
	for _, step := range steps {
		use, _ := step["use"].(string)
		if use == validationTask {
			continue
		}
		id, _ := step["id"].(string)
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("FCAF scenario %s contains a step without id", path)
		}
		if _, exists := ids[id]; exists {
			return nil, fmt.Errorf("FCAF scenario %s contains duplicate step id %q", path, id)
		}
		ids[id] = prefix + "-" + id
	}
	return ids, nil
}

func mergeValidationStep(
	path string,
	step map[string]any,
	stepIDs map[string]string,
	fixture map[string]string,
	pipelineOutputs map[string]any,
	testIDs map[string]struct{},
	allowedTestIDs map[string]struct{},
) error {
	with, ok := step["with"].(map[string]any)
	if !ok {
		return fmt.Errorf("FCAF validation step in %s has no with object", path)
	}
	addTestID := func(id string) {
		if id == "" {
			return
		}
		if allowedTestIDs != nil {
			if _, allowed := allowedTestIDs[id]; !allowed {
				return
			}
		}
		testIDs[id] = struct{}{}
	}
	for _, id := range walletroles.StringList(with["test_ids"]) {
		addTestID(id)
	}
	if id, _ := with["test_id"].(string); id != "" {
		addTestID(id)
	}
	rawOutputs, ok := with["pipeline_outputs"].(map[string]any)
	if !ok || len(rawOutputs) == 0 {
		return fmt.Errorf("FCAF validation step in %s has no pipeline_outputs", path)
	}
	rewritten := walletroles.OptionalizeExpressions(
		rewriteValue(rawOutputs, stepIDs, fixture),
	).(map[string]any)
	for source, output := range rewritten {
		if previous, exists := pipelineOutputs[source]; exists {
			return fmt.Errorf(
				"FCAF evidence source %q is defined by multiple scenarios (%T and %s)",
				source,
				previous,
				path,
			)
		}
		pipelineOutputs[source] = output
	}
	return nil
}

func rewriteValue(value any, stepIDs map[string]string, fixture map[string]string) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = rewriteValue(child, stepIDs, fixture)
		}
		if id, ok := out["id"].(string); ok {
			if rewritten, exists := stepIDs[id]; exists {
				out["id"] = rewritten
			}
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, child := range typed {
			out[index] = rewriteValue(child, stepIDs, fixture)
		}
		return out
	case string:
		return rewriteString(typed, stepIDs, fixture)
	default:
		return value
	}
}

func rewriteString(value string, stepIDs map[string]string, fixture map[string]string) string {
	keys := make([]string, 0, len(stepIDs))
	for key := range stepIDs {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, oldID := range keys {
		value = strings.ReplaceAll(value, "${{ "+oldID+".", "${{ "+stepIDs[oldID]+".")
		if value == oldID {
			value = stepIDs[oldID]
		}
	}
	return walletroles.SubstituteFixture(value, fixture)
}

// declaredFixture returns the string values of a scenario's runtime.fixture.
// The generated pipeline keeps the other ${fixture.<key>} tokens, which
// pipeline.ApplyFixture fills with its defaults at run time.
func declaredFixture(runtime map[string]any) map[string]string {
	declared := nestedMap(runtime, "fixture")
	fixture := make(map[string]string, len(declared))
	for key, value := range declared {
		if text, ok := value.(string); ok {
			fixture[key] = text
		}
	}
	return fixture
}

func nestedMap(values map[string]any, key string) map[string]any {
	if values == nil {
		return nil
	}
	value, _ := values[key].(map[string]any)
	return value
}

// ScenarioPrefix is the prefix of the step IDs a scenario file's steps get
// in an aggregate pipeline of suite: the file name without the suite's
// prefix, cut to 36 characters, and 8 hex digits of its sha256.
func ScenarioPrefix(suite string, path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	name = strings.TrimPrefix(name, "fcaf-"+SuiteSlug(suite)+"-")
	name = nonSlugCharacter.ReplaceAllString(strings.ToLower(name), "-")
	name = strings.Trim(name, "-")
	if len(name) > 36 {
		name = strings.Trim(name[:36], "-")
	}
	sum := sha256.Sum256([]byte(filepath.Base(path)))
	return fmt.Sprintf("%s-%x", name, sum[:4])
}
