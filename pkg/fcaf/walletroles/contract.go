// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package walletroles loads the FCAF wallet role contract: what every wallet
// action the FCAF suites run must do, whatever the wallet.
package walletroles

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// FileName is the contract file under the FCAF templates root.
const FileName = "fcaf-roles.yaml"

// Outcome is what decides whether a role's step fails.
type Outcome string

const (
	// OutcomeSuccess fails the step unless the wallet completes the flow.
	OutcomeSuccess Outcome = "strict-success"
	// OutcomeRejection fails the step when the wallet accepts or completes.
	OutcomeRejection Outcome = "strict-rejection"
	// OutcomeTolerant never fails the step on the outcome; protocol evidence
	// decides.
	OutcomeTolerant Outcome = "tolerant"
	// OutcomeWait only keeps the wallet idle for WaitSeconds.
	OutcomeWait Outcome = "wait"
)

// App is an app a role's flow drives.
type App string

// The apps of the reference app_ids.
const (
	AppWallet  App = "wallet"
	AppBrowser App = "browser"
)

// ScreenshotCondition is when a flow stores a screenshot.
type ScreenshotCondition string

const (
	// ScreenshotGated is top-level, after a non-optional check passed.
	ScreenshotGated ScreenshotCondition = "gated"
	// ScreenshotWhileScreen is inside a runFlow with when:.
	ScreenshotWhileScreen ScreenshotCondition = "while-screen"
	// ScreenshotUnconditional is top-level, before any non-optional check.
	ScreenshotUnconditional ScreenshotCondition = "unconditional"
)

// Injection marks the roles the pipeline generator runs on its own.
type Injection string

// The injected roles: the wallet reset that starts every scenario, and the PID
// issuance run before a presentation that needs a credential.
const (
	InjectedReset    Injection = "reset"
	InjectedIssuance Injection = "issuance"
)

// Screenshot is a verdict screenshot FCAF tests read by name.
type Screenshot struct {
	Name   string              `yaml:"name"             json:"name"`
	When   ScreenshotCondition `yaml:"when"             json:"when"`
	Screen string              `yaml:"screen,omitempty" json:"screen,omitempty"`
}

// Role is one wallet action the FCAF suites run.
type Role struct {
	Name                 string       `yaml:"name"                        json:"name"`
	Summary              string       `yaml:"summary"                     json:"summary"`
	Suites               []string     `yaml:"suites"                      json:"suites"`
	Outcome              Outcome      `yaml:"outcome"                     json:"outcome"`
	FailsWhen            string       `yaml:"fails_when"                  json:"fails_when"`
	WaitSeconds          int          `yaml:"wait_seconds,omitempty"      json:"wait_seconds,omitempty"`
	Parameters           []string     `yaml:"parameters"                  json:"parameters"`
	HeaderApp            App          `yaml:"header_app"                  json:"header_app"`
	LaunchApps           []App        `yaml:"launch_apps"                 json:"launch_apps"`
	Presents             int          `yaml:"presents"                    json:"presents"`
	IssuesCredential     bool         `yaml:"issues_credential,omitempty" json:"issues_credential,omitempty"`
	AbsenceProbe         bool         `yaml:"absence_probe,omitempty"     json:"absence_probe,omitempty"`
	Injected             Injection    `yaml:"injected,omitempty"          json:"injected,omitempty"`
	Silent               bool         `yaml:"silent,omitempty"            json:"silent,omitempty"`
	DiagnosticNameBudget int          `yaml:"diagnostic_name_budget"      json:"diagnostic_name_budget"`
	Screenshots          []Screenshot `yaml:"screenshots"                 json:"screenshots"`
	// Category is the wallet action category of the reference manifest.
	Category string `yaml:"-" json:"category"`
	// ReferenceCode is the reference flow source.
	ReferenceCode string `yaml:"-" json:"reference_code"`
}

// Reference names the reference wallet whose flows implement every role.
type Reference struct {
	// Manifest is relative to the FCAF templates root.
	Manifest string         `yaml:"manifest" json:"manifest"`
	AppIDs   map[App]string `yaml:"app_ids"  json:"app_ids"`
}

// Contract is the decoded fcaf-roles.yaml.
type Contract struct {
	Version   int       `yaml:"version"   json:"version"`
	Reference Reference `yaml:"reference" json:"reference"`
	Rules     []string  `yaml:"rules"     json:"rules"`
	Roles     []Role    `yaml:"roles"     json:"roles"`
	// ReferencePrefix is "<organization>/<wallet>/" of the reference
	// manifest: the action_id prefix of the reference flows in scenarios.
	ReferencePrefix string `yaml:"-" json:"-"`

	// fixturesMu guards fixtureSuites, the scenarios Fixtures reads per
	// (fcafRoot, suite).
	fixturesMu    sync.Mutex
	fixtureSuites map[suiteKey]*suiteScenarios
}

// SHA256 is the lowercase hex sha256 of the role's JSON encoding: every
// field counts, so an edited contract entry or reference flow changes it.
func (r Role) SHA256() string {
	data, err := json.Marshal(r)
	if err != nil {
		// A Role holds only strings, ints, bools and slices of them.
		panic(fmt.Sprintf("walletroles: encode role %s: %v", r.Name, err))
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// StartsFrom returns the names of the roles a role's action starts from, in
// order: none for onboarding, the reset for the others, and the reset then
// the PID issuance for presentations, as the generator runs every scenario.
// An absence probe starts from the reset alone: the generator injects no PID
// before it.
func (c *Contract) StartsFrom(role Role) []string {
	switch {
	case role.Category == "onboarding":
		return []string{}
	case role.Category == "verify-credential" && !role.AbsenceProbe:
		return []string{c.Injected(InjectedReset).Name, c.Injected(InjectedIssuance).Name}
	default:
		return []string{c.Injected(InjectedReset).Name}
	}
}

var (
	roleNamePattern       = regexp.MustCompile(`^[a-z0-9-]+$`)
	parameterPattern      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	screenshotNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	knownSuites           = []string{
		"wallet_solution/relying_party",
		"wallet_solution/attestation_provider",
	}
)

// Load reads and validates the contract under fcafRoot and fills every role's
// Category and ReferenceCode from the reference manifest.
func Load(fcafRoot string) (*Contract, error) {
	path := filepath.Join(fcafRoot, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fcaf roles: %w", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var contract Contract
	if err := decoder.Decode(&contract); err != nil {
		return nil, fmt.Errorf("fcaf roles: parse %s: %w", path, err)
	}
	if err := contract.validate(); err != nil {
		return nil, fmt.Errorf("fcaf roles: %w", err)
	}
	if err := contract.loadReference(fcafRoot); err != nil {
		return nil, fmt.Errorf("fcaf roles: %w", err)
	}
	return &contract, nil
}

// Role returns the role of that name.
func (c *Contract) Role(name string) (Role, bool) {
	for _, role := range c.Roles {
		if role.Name == name {
			return role, true
		}
	}
	return Role{}, false
}

// Injected returns the role of that injection kind; Load guarantees exactly
// one per kind.
func (c *Contract) Injected(kind Injection) Role {
	for _, role := range c.Roles {
		if role.Injected == kind {
			return role
		}
	}
	return Role{}
}

func (c *Contract) validate() error {
	if c.Version != 1 {
		return fmt.Errorf("version must be 1, got %d", c.Version)
	}
	if strings.TrimSpace(c.Reference.Manifest) == "" {
		return errors.New("reference.manifest is required")
	}
	if len(c.Reference.AppIDs) != 2 ||
		c.Reference.AppIDs[AppWallet] == "" || c.Reference.AppIDs[AppBrowser] == "" {
		return errors.New("reference.app_ids must name exactly wallet and browser")
	}
	names := map[string]struct{}{}
	injected := map[Injection]int{}
	for index, role := range c.Roles {
		if !roleNamePattern.MatchString(role.Name) {
			return fmt.Errorf("role %d: name %q must match %s", index, role.Name, roleNamePattern)
		}
		if _, duplicate := names[role.Name]; duplicate {
			return fmt.Errorf("role %q: name is duplicated", role.Name)
		}
		names[role.Name] = struct{}{}
		if err := role.validate(); err != nil {
			return fmt.Errorf("role %q: %w", role.Name, err)
		}
		if role.Injected != "" {
			injected[role.Injected]++
		}
	}
	for _, kind := range []Injection{InjectedReset, InjectedIssuance} {
		if injected[kind] != 1 {
			return fmt.Errorf("exactly one role must be injected: %s, got %d", kind, injected[kind])
		}
	}
	return nil
}

func (r Role) validate() error {
	if strings.TrimSpace(r.Summary) == "" {
		return errors.New("summary is required")
	}
	if strings.TrimSpace(r.FailsWhen) == "" {
		return errors.New("fails_when is required")
	}
	if len(r.Suites) == 0 {
		return errors.New("suites is required")
	}
	if err := uniqueIn("suites", r.Suites, func(suite string) bool {
		return slices.Contains(knownSuites, suite)
	}); err != nil {
		return err
	}
	switch r.Outcome {
	case OutcomeSuccess, OutcomeRejection, OutcomeTolerant, OutcomeWait:
	default:
		return fmt.Errorf("outcome %q is unknown", r.Outcome)
	}
	if r.Outcome == OutcomeWait {
		if r.WaitSeconds < 1 || r.WaitSeconds > 600 {
			return fmt.Errorf("wait_seconds must be 1-600, got %d", r.WaitSeconds)
		}
	} else if r.WaitSeconds != 0 {
		return errors.New("wait_seconds is only for the wait outcome")
	}
	if err := uniqueIn("parameters", r.Parameters, parameterPattern.MatchString); err != nil {
		return err
	}
	if !knownApp(r.HeaderApp) {
		return fmt.Errorf("header_app %q is unknown", r.HeaderApp)
	}
	if err := uniqueIn("launch_apps", r.LaunchApps, knownApp); err != nil {
		return err
	}
	if r.Presents < 0 {
		return errors.New("presents must not be negative")
	}
	if r.IssuesCredential && r.Presents != 0 {
		return errors.New("a role that issues a credential presents none")
	}
	switch r.Injected {
	case "", InjectedReset, InjectedIssuance:
	default:
		return fmt.Errorf("injected %q is unknown", r.Injected)
	}
	if r.Injected != "" && !r.Silent {
		return errors.New("an injected role must be silent")
	}
	if r.Silent && len(r.Screenshots) > 0 {
		return errors.New("a silent role takes no screenshot")
	}
	if r.DiagnosticNameBudget < 1 || r.DiagnosticNameBudget > 99 {
		return fmt.Errorf("diagnostic_name_budget must be 1-99, got %d", r.DiagnosticNameBudget)
	}
	shots := map[string]struct{}{}
	for _, shot := range r.Screenshots {
		if !screenshotNamePattern.MatchString(shot.Name) {
			return fmt.Errorf("screenshot name %q must match %s", shot.Name, screenshotNamePattern)
		}
		if _, duplicate := shots[shot.Name]; duplicate {
			return fmt.Errorf("screenshot %q is duplicated", shot.Name)
		}
		shots[shot.Name] = struct{}{}
		switch shot.When {
		case ScreenshotGated, ScreenshotWhileScreen, ScreenshotUnconditional:
		default:
			return fmt.Errorf("screenshot %q: when %q is unknown", shot.Name, shot.When)
		}
		if shot.When != ScreenshotUnconditional && strings.TrimSpace(shot.Screen) == "" {
			return fmt.Errorf("screenshot %q: screen is required when %s", shot.Name, shot.When)
		}
	}
	return nil
}

func knownApp(app App) bool {
	return app == AppWallet || app == AppBrowser
}

func uniqueIn[T ~string](field string, values []T, valid func(T) bool) error {
	seen := map[T]struct{}{}
	for _, value := range values {
		if !valid(value) {
			return fmt.Errorf("%s: %q is not valid", field, value)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%s: %q is duplicated", field, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func (c *Contract) loadReference(fcafRoot string) error {
	manifestPath := filepath.Join(fcafRoot, c.Reference.Manifest)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read reference manifest: %w", err)
	}
	type manifestAction struct {
		Name     string `yaml:"name"`
		Category string `yaml:"category"`
		File     string `yaml:"file"`
	}
	var manifest struct {
		Organization string           `yaml:"organization"`
		Wallet       string           `yaml:"wallet"`
		Actions      []manifestAction `yaml:"actions"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse reference manifest %s: %w", manifestPath, err)
	}
	if manifest.Organization == "" || manifest.Wallet == "" {
		return fmt.Errorf(
			"reference manifest %s must name organization and wallet",
			c.Reference.Manifest,
		)
	}
	c.ReferencePrefix = manifest.Organization + "/" + manifest.Wallet + "/"
	for index := range c.Roles {
		role := &c.Roles[index]
		entry := slices.IndexFunc(manifest.Actions, func(action manifestAction) bool {
			return action.Name == role.Name
		})
		if entry < 0 {
			return fmt.Errorf("role %q has no entry in %s", role.Name, c.Reference.Manifest)
		}
		action := manifest.Actions[entry]
		role.Category = action.Category
		code, err := os.ReadFile(filepath.Join(filepath.Dir(manifestPath), action.File))
		if err != nil {
			return fmt.Errorf("role %q: read reference flow: %w", role.Name, err)
		}
		role.ReferenceCode = string(code)
	}
	return nil
}

// strictChecks are the commands that fail a flow when their condition does
// not hold, unless they are optional.
var strictChecks = []string{
	"assertVisible",
	"assertNotVisible",
	"assertTrue",
	"scrollUntilVisible",
	"extendedWaitUntil",
}

// ScreenshotConditions returns, for every screenshot a Maestro flow stores,
// the condition of each takeScreenshot command of that name.
func ScreenshotConditions(code string) (map[string][]ScreenshotCondition, error) {
	commands, err := flowCommandList(code)
	if err != nil {
		return nil, err
	}
	conditions := map[string][]ScreenshotCondition{}
	strict := false
	walkScreenshots(commands, &strict, false, conditions)
	return conditions, nil
}

// flowCommands returns the command list of a Maestro flow: the second YAML
// document, after the header.
func flowCommands(code string) ([]any, error) {
	list, err := flowCommandList(code)
	if err != nil {
		return nil, err
	}
	var commands []any
	if err := list.Decode(&commands); err != nil {
		return nil, fmt.Errorf("parse Maestro flow: %w", err)
	}
	return commands, nil
}

// flowCommandList returns the YAML sequence node of a Maestro flow's
// commands: the second YAML document, after the header.
func flowCommandList(code string) (*yaml.Node, error) {
	decoder := yaml.NewDecoder(strings.NewReader(code))
	var documents []*yaml.Node
	for {
		var document yaml.Node
		err := decoder.Decode(&document)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse Maestro flow: %w", err)
		}
		documents = append(documents, &document)
	}
	if len(documents) < 2 {
		return nil, errors.New("the Maestro flow has no command list after its header")
	}
	commands := resolveNode(documents[1])
	if commands == nil || commands.Kind != yaml.SequenceNode {
		return nil, errors.New("the Maestro flow's commands are not a list")
	}
	return commands, nil
}

// resolveNode unwraps document and alias nodes; nil for an empty document.
func resolveNode(node *yaml.Node) *yaml.Node {
	for node != nil {
		switch node.Kind {
		case yaml.DocumentNode:
			if len(node.Content) == 0 {
				return nil
			}
			node = node.Content[0]
		case yaml.AliasNode:
			node = node.Alias
		default:
			return node
		}
	}
	return nil
}

// mappingValue returns the value of key in a mapping node, or nil.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return resolveNode(node.Content[index+1])
		}
	}
	return nil
}

// nestedCommands returns the commands list of a runFlow, retry or repeat.
func nestedCommands(options *yaml.Node) *yaml.Node {
	commands := mappingValue(options, "commands")
	if commands == nil || commands.Kind != yaml.SequenceNode {
		return nil
	}
	return commands
}

// isPlainFalse reports whether node is the unquoted YAML boolean false, with or without an
// explicit !!bool tag (the Python validator cannot tell the two apart).
func isPlainFalse(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode && node.Tag == "!!bool" &&
		node.Style&^yaml.TaggedStyle == 0 && node.Value == "false"
}

// isString reports whether node is a YAML string scalar.
func isString(node *yaml.Node) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Tag == "!!str"
}

// strictCheckHolds reports whether a strict check's options make it
// non-optional: no options mapping, no optional key, or optional: false
// written as the plain boolean.
func strictCheckHolds(options *yaml.Node) bool {
	if options == nil || options.Kind != yaml.MappingNode {
		return true
	}
	optional := mappingValue(options, "optional")
	return optional == nil || isPlainFalse(optional)
}

func walkScreenshots(
	commands *yaml.Node,
	strict *bool,
	inWhen bool,
	conditions map[string][]ScreenshotCondition,
) {
	if commands == nil {
		return
	}
	for _, raw := range commands.Content {
		command := resolveNode(raw)
		if command == nil || command.Kind != yaml.MappingNode {
			continue
		}
		for index := 0; index+1 < len(command.Content); index += 2 {
			key := command.Content[index].Value
			value := resolveNode(command.Content[index+1])
			switch {
			case key == "takeScreenshot":
				name := ""
				if isString(value) {
					name = value.Value
				} else if path := mappingValue(value, "path"); isString(path) {
					name = path.Value
				}
				condition := ScreenshotUnconditional
				switch {
				case inWhen:
					condition = ScreenshotWhileScreen
				case *strict:
					condition = ScreenshotGated
				}
				conditions[name] = append(conditions[name], condition)
			case key == "runFlow":
				nested := nestedCommands(value)
				if mappingValue(value, "when") != nil {
					branch := *strict
					walkScreenshots(nested, &branch, true, conditions)
					continue
				}
				walkScreenshots(nested, strict, inWhen, conditions)
			case key == "retry":
				walkScreenshots(nestedCommands(value), strict, inWhen, conditions)
			case key == "repeat":
				// A repeat body may never run, so its checks prove nothing
				// for later commands.
				body := *strict
				walkScreenshots(nestedCommands(value), &body, inWhen, conditions)
			case slices.Contains(strictChecks, key):
				if !inWhen && strictCheckHolds(value) {
					*strict = true
				}
			}
		}
	}
}
