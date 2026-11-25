// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package walletroles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/internal/pipelineexpr"
	"gopkg.in/yaml.v3"
)

// PID issuers of Capture the FCAF action author may issue the PID from.
const (
	PIDIssuerKeyAttestation = "key-attestation" // eu-pid-device-bound, key attestation required
	PIDIssuerJWTProof       = "jwt-proof"       // eu-pid-jwt-proof-only, plain JWT proof
)

// PIDIssuers lists them, the one tried first first.
var PIDIssuers = []string{PIDIssuerKeyAttestation, PIDIssuerJWTProof}

// pidIssuerConfigurations maps each PID issuer to its Capture issuer
// configuration and the suffix of its credential configurations.
var pidIssuerConfigurations = map[string]struct {
	issuerConfigurationID string
	credentialSuffix      string
}{
	PIDIssuerKeyAttestation: {"eu-pid-device-bound", "key-attestation-required"},
	PIDIssuerJWTProof:       {"eu-pid-jwt-proof-only", "jwt-proof"},
}

// The PID formats of the injected issuance fixtures.
const (
	pidFormatSDJWT = "sd-jwt"
	pidFormatMdoc  = "mdoc"
)

// OpenID4VP request schemes of the FCAF verifier session requests. The
// scenarios ask for HAIPRequestScheme; a wallet that registers only
// OpenID4VPRequestScheme gets its requests rewritten by WithRequestScheme.
const (
	HAIPRequestScheme      = "haip-vp://"
	OpenID4VPRequestScheme = "openid4vp://"
)

const (
	mobileAutomationUse = "mobile-automation"
	httpRequestUse      = "http-request"
	fcafValidationUse   = "fcaf-validation"
	outputsMarker       = ".outputs"
)

// FixtureRequest is the HTTP request that creates the session a role's
// parameter opens.
type FixtureRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`
}

// FixtureFetch is a GET step that reads back the session the request created.
type FixtureFetch struct {
	StepID string
	// URL contains ${session_id}.
	URL string
}

// Fixture is one way to exercise a role the way an FCAF scenario does: the
// request that creates its parameter, the fetches that read the evidence
// back, and the tests that evidence feeds.
type Fixture struct {
	// ID is "<scenario base name>#<mobile step id>", or "pid-<format>" /
	// "pid-jwt-proof-<format>" for the injected issuance.
	ID string
	// Scenario is the scenario file base name without .yaml; "" for PID
	// fixtures.
	Scenario     string
	MobileStepID string
	// Standalone is set when the role's step is its scenario's first
	// mobile-automation step.
	Standalone bool
	// PIDFormat is "sd-jwt", "mdoc" or "".
	PIDFormat string
	// PIDIssuer is one of PIDIssuers for issuance fixtures, "" otherwise.
	PIDIssuer       string
	Parameter       string
	Request         FixtureRequest
	ValuePath       string
	Fetches         []FixtureFetch
	TestIDs         []string
	PipelineOutputs map[string]any
	// Setup lists what the start state must hold beyond the role's starting actions, in order:
	// a PID of another format for each further format the request asks for that no earlier
	// offer issues, and the scenario's own earlier credential offers a non-standalone step
	// depends on.
	Setup []FixtureSetup `json:"setup"`
	// Complete is set when Setup reproduces the start state: the fixture is standalone, or
	// every earlier step that leaves a credential in the wallet is one of its offers.
	Complete bool `json:"complete"`
	// Class is the fixture's behaviour class: what the wallet can show for its
	// request, built from the contract's authoring_class_fields (Contract.Class).
	Class string `json:"class"`
}

// FixtureSetup is one credential a fixture's start state holds beyond the
// role's starting actions: an injected PID of PIDFormat, or a scenario offer
// that Role accepted, whose parameter Request generates.
type FixtureSetup struct {
	Role      string          `json:"role"`              // issuance role, or the role that ran the offer
	PIDFormat string          `json:"pid_format"`        // set for an injected PID; the runner picks the wallet's issuer
	Request   *FixtureRequest `json:"request,omitempty"` // set for a scenario offer
	ValuePath string          `json:"value_path,omitempty"`
}

var (
	fixtureParameterSource = regexp.MustCompile(
		`^\$\{\{\s*([A-Za-z0-9_.-]+)\.outputs\.body\.([A-Za-z0-9_.]+)\s*\}\}$`,
	)
	fixtureHeaders = []string{"content-type", "accept", "accept-language"}
)

// PIDIssuance is the issuer session request of the injected PID issuance the
// generator runs before a relying-party presentation, from one of PIDIssuers
// in format "sd-jwt" or "mdoc". An unknown issuer panics: callers pass only
// constants.
func PIDIssuance(issuer, format string) FixtureRequest {
	configuration, ok := pidIssuerConfigurations[issuer]
	if !ok {
		panic(fmt.Sprintf("walletroles: unknown PID issuer %q", issuer))
	}
	return FixtureRequest{
		Method:  "POST",
		URL:     pipelineexpr.DefaultVerifierURL + "/sessions",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body: map[string]any{
			"issuer_configuration_id": configuration.issuerConfigurationID,
			"credential_configuration_id": "urn:eu.europa.ec.eudi:pid:1." + format + "." +
				configuration.credentialSuffix,
		},
	}
}

// OptionalizeExpressions marks every whole-value ${{ ... }} expression
// optional, so evidence whose step did not run resolves to nothing instead of
// failing the resolution.
func OptionalizeExpressions(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = OptionalizeExpressions(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, child := range typed {
			out[index] = OptionalizeExpressions(child)
		}
		return out
	case string:
		trimmed := strings.TrimSpace(typed)
		if strings.HasPrefix(trimmed, "${{") && strings.HasSuffix(trimmed, "}}") {
			inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "${{"), "}}"))
			if !strings.Contains(inner, "| optional") {
				return "${{ " + inner + " | optional }}"
			}
		}
		return typed
	default:
		return value
	}
}

// WithRequestScheme returns a deep copy of a fixture request body. When
// scheme is OpenID4VPRequestScheme, a top-level "scheme" equal to
// HAIPRequestScheme becomes OpenID4VPRequestScheme; any other value, or a
// missing key, is left unchanged.
func WithRequestScheme(body any, scheme string) any {
	out := deepCopy(body)
	if scheme != OpenID4VPRequestScheme {
		return out
	}
	if fields, ok := out.(map[string]any); ok && fields["scheme"] == HAIPRequestScheme {
		fields["scheme"] = OpenID4VPRequestScheme
	}
	return out
}

func deepCopy(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = deepCopy(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, child := range typed {
			out[index] = deepCopy(child)
		}
		return out
	default:
		return value
	}
}

// Scenario is an FCAF scenario file, decoded loosely.
type Scenario struct {
	Name    string           `yaml:"name"`
	Runtime map[string]any   `yaml:"runtime,omitempty"`
	Steps   []map[string]any `yaml:"steps"`
}

// LoadScenario reads the FCAF scenario at path, which must define a name and
// steps.
func LoadScenario(path string) (Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, fmt.Errorf("read FCAF scenario %s: %w", path, err)
	}
	var scenario Scenario
	if err := yaml.Unmarshal(data, &scenario); err != nil {
		return Scenario{}, fmt.Errorf("parse FCAF scenario %s: %w", path, err)
	}
	if strings.TrimSpace(scenario.Name) == "" || len(scenario.Steps) == 0 {
		return Scenario{}, fmt.Errorf("FCAF scenario %s must define name and steps", path)
	}
	return scenario, nil
}

// ScenarioFixture returns the ${fixture.<key>} values of a scenario: the
// defaults (pipelineexpr.FixtureDefaults), overridden by its runtime.fixture
// strings, as pipeline.ApplyFixture substitutes them at run time.
func ScenarioFixture(runtime map[string]any) map[string]string {
	values := pipelineexpr.FixtureDefaults()
	declared, _ := runtime["fixture"].(map[string]any)
	for key, value := range declared {
		if text, ok := value.(string); ok {
			values[key] = text
		}
	}
	return values
}

// SubstituteFixture replaces every ${fixture.<key>} token of s whose key
// fixture holds by its value.
func SubstituteFixture(s string, fixture map[string]string) string {
	for key, value := range fixture {
		s = strings.ReplaceAll(s, "${fixture."+key+"}", value)
	}
	return s
}

// suiteKey names the scenarios of a suite under an FCAF templates root.
type suiteKey struct{ root, suite string }

// suiteScenarios are the scenarios of a suite, parsed and with their fixture
// values substituted, and the loader of the suite's tests.
type suiteScenarios struct {
	scenarios []fixtureScenario
	tests     *testLoader
}

// fixtureScenario is a scenario's path and file base name without .yaml,
// and its steps with their fixture values substituted, also by step id.
type fixtureScenario struct {
	path  string
	name  string
	steps []map[string]any
	byID  map[string]map[string]any
}

// Fixtures returns the fixtures of a role in the scenarios of a suite under
// fcafRoot: every step of a scenario that runs the role's reference action
// with a parameter a self-contained POST request produces, each with its
// behaviour Class. Standalone fixtures come first; file and step order is
// kept otherwise. The contract reads the scenarios and tests of each
// (fcafRoot, suite) once.
func (c *Contract) Fixtures(fcafRoot, suite, roleName string) ([]Fixture, error) {
	role, ok := c.Role(roleName)
	if !ok {
		return nil, fmt.Errorf("role %q is not in the contract", roleName)
	}
	if role.Injected == InjectedIssuance {
		fixtures := pidFixtures(role)
		for i := range fixtures {
			fixtures[i].Class = c.Class(fixtures[i].Request.Body)
		}
		return fixtures, nil
	}
	if len(role.Parameters) == 0 {
		return []Fixture{}, nil
	}
	loaded, err := c.suiteScenarios(fcafRoot, suite)
	if err != nil {
		return nil, err
	}
	fixtures := []Fixture{}
	for _, scenario := range loaded.scenarios {
		found, err := c.scenarioFixtures(scenario, role, loaded.tests)
		if err != nil {
			return nil, err
		}
		fixtures = append(fixtures, found...)
	}
	sort.SliceStable(fixtures, func(i, j int) bool {
		return fixtures[i].Standalone && !fixtures[j].Standalone
	})
	for i := range fixtures {
		fixtures[i].Class = c.Class(fixtures[i].Request.Body)
	}
	return fixtures, nil
}

// suiteScenarios returns the scenarios of the suite under fcafRoot, read on
// first use; a failed read is not kept.
func (c *Contract) suiteScenarios(fcafRoot, suite string) (*suiteScenarios, error) {
	c.fixturesMu.Lock()
	defer c.fixturesMu.Unlock()
	key := suiteKey{root: fcafRoot, suite: suite}
	if loaded, ok := c.fixtureSuites[key]; ok {
		return loaded, nil
	}
	suiteRoot := filepath.Join(fcafRoot, filepath.FromSlash(suite))
	paths, err := filepath.Glob(filepath.Join(suiteRoot, "scenarios", "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("list FCAF scenarios of %s: %w", suite, err)
	}
	sort.Strings(paths)
	loaded := &suiteScenarios{
		scenarios: make([]fixtureScenario, 0, len(paths)),
		tests:     &testLoader{root: filepath.Join(suiteRoot, "tests")},
	}
	for _, path := range paths {
		scenario, err := LoadScenario(path)
		if err != nil {
			return nil, err
		}
		loaded.scenarios = append(loaded.scenarios, substitutedScenario(path, scenario))
	}
	if c.fixtureSuites == nil {
		c.fixtureSuites = map[suiteKey]*suiteScenarios{}
	}
	c.fixtureSuites[key] = loaded
	return loaded, nil
}

// substitutedScenario substitutes the scenario's fixture values in its steps.
func substitutedScenario(path string, scenario Scenario) fixtureScenario {
	values := ScenarioFixture(scenario.Runtime)
	parsed := fixtureScenario{
		path:  path,
		name:  strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		steps: make([]map[string]any, len(scenario.Steps)),
		byID:  make(map[string]map[string]any, len(scenario.Steps)),
	}
	for i, step := range scenario.Steps {
		substituted, _ := substituteFixture(step, values).(map[string]any)
		parsed.steps[i] = substituted
		if id, _ := substituted["id"].(string); id != "" {
			parsed.byID[id] = substituted
		}
	}
	return parsed
}

func pidFixtures(role Role) []Fixture {
	parameter := ""
	if len(role.Parameters) > 0 {
		parameter = role.Parameters[0]
	}
	fixtures := make([]Fixture, 0, len(PIDIssuers)*2)
	for _, issuer := range PIDIssuers {
		prefix := "pid-"
		if issuer != PIDIssuerKeyAttestation {
			prefix += issuer + "-"
		}
		for _, format := range []string{pidFormatSDJWT, pidFormatMdoc} {
			fixtures = append(fixtures, Fixture{
				ID:              prefix + format,
				PIDFormat:       format,
				PIDIssuer:       issuer,
				Parameter:       parameter,
				Request:         PIDIssuance(issuer, format),
				ValuePath:       "deeplink",
				Fetches:         []FixtureFetch{},
				TestIDs:         []string{},
				PipelineOutputs: map[string]any{},
				Setup:           []FixtureSetup{},
				Complete:        true,
			})
		}
	}
	return fixtures
}

// testLoader loads the suite's test definitions once, on first use; a
// failed load is tried again.
type testLoader struct {
	root   string
	mu     sync.Mutex
	tests  map[string]dsl.TestDefinition
	loaded bool
}

func (l *testLoader) load() (map[string]dsl.TestDefinition, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.loaded {
		return l.tests, nil
	}
	tests, err := catalog.LoadTests(l.root)
	if err != nil {
		return nil, fmt.Errorf("load FCAF tests %s: %w", l.root, err)
	}
	l.tests, l.loaded = tests, true
	return tests, nil
}

func (c *Contract) scenarioFixtures(
	scenario fixtureScenario,
	role Role,
	loader *testLoader,
) ([]Fixture, error) {
	actionID := c.ReferencePrefix + role.Name
	var fixtures []Fixture
	mobileSeen := false
	for index, step := range scenario.steps {
		use, _ := step["use"].(string)
		if use != mobileAutomationUse {
			continue
		}
		standalone := !mobileSeen
		mobileSeen = true
		with, _ := step["with"].(map[string]any)
		if id, _ := with["action_id"].(string); id != actionID {
			continue
		}
		fixture, ok := stepFixture(step, role, scenario.byID, scenario.steps)
		if !ok {
			continue
		}
		fixture.ID = scenario.name + "#" + fixture.MobileStepID
		fixture.Scenario = scenario.name
		fixture.Standalone = standalone
		fixture.Setup, fixture.Complete = c.fixtureSetup(
			fixture,
			role,
			scenario,
			scenario.steps[:index],
		)
		if err := scenarioTests(&fixture, scenario.steps, loader); err != nil {
			return nil, fmt.Errorf("FCAF scenario %s: %w", scenario.path, err)
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures, nil
}

// fixtureSetup returns the setup of a fixture whose role step comes after
// earlier, and whether it reproduces the fixture's start state. For a
// non-standalone fixture it lists the offers of earlier steps whose role issues
// a credential and is not injected, when a self-contained POST generates their
// parameter; offers of a strict-rejection role or of the fixture's own role
// leave nothing a conforming wallet holds and are skipped. For a role that
// starts from the injected PID, a PID of each further format its request asks
// for and no offer issues comes first. An issuing step whose offer cannot be
// reproduced makes the setup incomplete.
func (c *Contract) fixtureSetup(
	fixture Fixture,
	role Role,
	scenario fixtureScenario,
	earlier []map[string]any,
) ([]FixtureSetup, bool) {
	offers := []FixtureSetup{}
	offered := map[string]bool{}
	complete := true
	if !fixture.Standalone {
		for _, step := range earlier {
			if use, _ := step["use"].(string); use != mobileAutomationUse {
				continue
			}
			with, _ := step["with"].(map[string]any)
			actionID, _ := with["action_id"].(string)
			name, isReference := strings.CutPrefix(actionID, c.ReferencePrefix)
			offerRole, known := c.Role(name)
			if !isReference || !known || !offerRole.IssuesCredential || offerRole.Injected != "" ||
				offerRole.Outcome == OutcomeRejection || offerRole.Name == role.Name {
				continue
			}
			_, valuePath, request, ok := stepRequest(step, offerRole, scenario.byID)
			if !ok {
				complete = false
				continue
			}
			offers = append(offers, FixtureSetup{
				Role:      offerRole.Name,
				Request:   &request,
				ValuePath: valuePath,
			})
			offered[offerFormat(request.Body)] = true
		}
	}
	setup := []FixtureSetup{}
	issuance := c.Injected(InjectedIssuance).Name
	if slices.Contains(c.StartsFrom(role), issuance) {
		for _, format := range RequestPIDFormats(fixture.Request.Body) {
			if format != fixture.PIDFormat && !offered[format] {
				setup = append(setup, FixtureSetup{Role: issuance, PIDFormat: format})
			}
		}
	}
	return append(setup, offers...), complete
}

// offerFormat is the format of the credential an offer request issues: mdoc
// when its credential_configuration_id or credential_id names mdoc, sd-jwt
// otherwise, as the pipeline generator counts issued credentials.
func offerFormat(body any) string {
	request, _ := body.(map[string]any)
	for _, key := range []string{"credential_configuration_id", "credential_id"} {
		if identifier, _ := request[key].(string); strings.Contains(
			strings.ToLower(identifier),
			"mdoc",
		) {
			return pidFormatMdoc
		}
	}
	return pidFormatSDJWT
}

func substituteFixture(value any, values map[string]string) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = substituteFixture(child, values)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, child := range typed {
			out[index] = substituteFixture(child, values)
		}
		return out
	case string:
		return SubstituteFixture(typed, values)
	default:
		return value
	}
}

// stepFixture builds the fixture of a mobile step whose only parameter comes
// from a self-contained POST request, mirroring the generator's
// deeplinkSourceStep.
func stepFixture(
	step map[string]any,
	role Role,
	byID map[string]map[string]any,
	steps []map[string]any,
) (Fixture, bool) {
	createID, valuePath, request, ok := stepRequest(step, role, byID)
	if !ok {
		return Fixture{}, false
	}
	stepID, _ := step["id"].(string)
	return Fixture{
		MobileStepID: stepID,
		PIDFormat:    requestPIDFormat(request.Body),
		Parameter:    role.Parameters[0],
		Request:      request,
		ValuePath:    valuePath,
		Fetches:      sessionFetches(createID, steps),
		TestIDs:      []string{},
		Setup:        []FixtureSetup{},
	}, true
}

// stepRequest returns the step that creates a mobile step's only parameter,
// the path of the parameter's value in its response body, and its request,
// when that request is a self-contained POST.
func stepRequest(
	step map[string]any,
	role Role,
	byID map[string]map[string]any,
) (string, string, FixtureRequest, bool) {
	with, _ := step["with"].(map[string]any)
	parameters, _ := with["parameters"].(map[string]any)
	if len(parameters) != 1 || len(role.Parameters) != 1 {
		return "", "", FixtureRequest{}, false
	}
	text, _ := parameters[role.Parameters[0]].(string)
	match := fixtureParameterSource.FindStringSubmatch(text)
	if match == nil {
		return "", "", FixtureRequest{}, false
	}
	createID, valuePath := match[1], match[2]
	request, ok := fixtureRequest(byID[createID])
	if !ok {
		return "", "", FixtureRequest{}, false
	}
	return createID, valuePath, request, true
}

// fixtureRequest returns the request of a POST http-request step that needs
// nothing but the fixture values, with only content negotiation headers.
func fixtureRequest(step map[string]any) (FixtureRequest, bool) {
	if use, _ := step["use"].(string); use != httpRequestUse {
		return FixtureRequest{}, false
	}
	with, _ := step["with"].(map[string]any)
	method, _ := with["method"].(string)
	if !strings.EqualFold(method, "POST") {
		return FixtureRequest{}, false
	}
	url, _ := with["url"].(string)
	body := with["body"]
	encoded, err := json.Marshal(body)
	if err != nil || url == "" || unresolved(url) || unresolved(string(encoded)) {
		return FixtureRequest{}, false
	}
	rawHeaders, _ := with["headers"].(map[string]any)
	var headers map[string]string
	for key, value := range rawHeaders {
		text, isString := value.(string)
		if !isString || !slices.Contains(fixtureHeaders, strings.ToLower(key)) {
			return FixtureRequest{}, false
		}
		if headers == nil {
			headers = make(map[string]string, len(rawHeaders))
		}
		headers[key] = text
	}
	return FixtureRequest{Method: "POST", URL: url, Headers: headers, Body: body}, true
}

func unresolved(text string) bool {
	return strings.Contains(text, "${{") || strings.Contains(text, "${fixture.")
}

// sessionFetches returns the GET steps that read back the session createID
// made, by its session_id and nothing else.
func sessionFetches(createID string, steps []map[string]any) []FixtureFetch {
	sessionRef := createID + ".outputs.body.session_id"
	expression := regexp.MustCompile(`\$\{\{\s*` + regexp.QuoteMeta(sessionRef) + `\s*\}\}`)
	fetches := []FixtureFetch{}
	for _, step := range steps {
		if use, _ := step["use"].(string); use != httpRequestUse {
			continue
		}
		with, _ := step["with"].(map[string]any)
		method, _ := with["method"].(string)
		url, _ := with["url"].(string)
		if !strings.EqualFold(method, "GET") || strings.Count(url, "${{") != 1 {
			continue
		}
		if refs := pipelineexpr.ExpressionRefs(url); len(refs) != 1 || refs[0] != sessionRef {
			continue
		}
		id, _ := step["id"].(string)
		fetches = append(fetches, FixtureFetch{
			StepID: id,
			URL:    expression.ReplaceAllLiteralString(url, "${session_id}"),
		})
	}
	return fetches
}

// requestCredentialQueries returns the DCQL credential queries of a
// presentation request body.
func requestCredentialQueries(body any) []any {
	request, _ := body.(map[string]any)
	presentation, _ := request["presentation_request"].(map[string]any)
	query, _ := presentation["dcql_query"].(map[string]any)
	credentials, _ := query["credentials"].([]any)
	return credentials
}

// pidVCT is the SD-JWT PID vct of the FCAF scenarios and PID issuers.
const pidVCT = "urn:eudi:pid:1"

// RequestPIDFormats returns the PID formats a presentation request body asks
// for, in sd-jwt, mdoc order: mdoc when a credential query is mso_mdoc,
// sd-jwt when one is dc+sd-jwt whose vct_values hold the PID vct or name none.
func RequestPIDFormats(body any) []string {
	sdJWT, mdoc := false, false
	for _, credential := range requestCredentialQueries(body) {
		entry, _ := credential.(map[string]any)
		switch format, _ := entry["format"].(string); format {
		case "mso_mdoc":
			mdoc = true
		case "dc+sd-jwt":
			meta, _ := entry["meta"].(map[string]any)
			values := StringList(meta["vct_values"])
			sdJWT = sdJWT || len(values) == 0 || slices.Contains(values, pidVCT)
		}
	}
	formats := []string{}
	if sdJWT {
		formats = append(formats, pidFormatSDJWT)
	}
	if mdoc {
		formats = append(formats, pidFormatMdoc)
	}
	return formats
}

// requestPIDFormat is the PID format a presentation request asks for: mdoc
// when any credential query is mso_mdoc, else sd-jwt when any is dc+sd-jwt.
func requestPIDFormat(body any) string {
	credentials := requestCredentialQueries(body)
	formats := make([]string, 0, len(credentials))
	for _, credential := range credentials {
		entry, _ := credential.(map[string]any)
		format, _ := entry["format"].(string)
		formats = append(formats, format)
	}
	switch {
	case slices.Contains(formats, "mso_mdoc"):
		return pidFormatMdoc
	case slices.Contains(formats, "dc+sd-jwt"):
		return pidFormatSDJWT
	default:
		return ""
	}
}

// scenarioTests keeps the scenario's tests whose evidence comes only from the
// fixture's mobile step and fetches, with the pipeline outputs they read.
func scenarioTests(fixture *Fixture, steps []map[string]any, loader *testLoader) error {
	allowed := map[string]struct{}{fixture.MobileStepID: {}}
	for _, fetch := range fixture.Fetches {
		allowed[fetch.StepID] = struct{}{}
	}
	fixture.PipelineOutputs = map[string]any{}
	for _, step := range steps {
		if use, _ := step["use"].(string); use != fcafValidationUse {
			continue
		}
		with, _ := step["with"].(map[string]any)
		ids := StringList(with["test_ids"])
		if id, _ := with["test_id"].(string); id != "" {
			ids = append(ids, id)
		}
		outputs, _ := with["pipeline_outputs"].(map[string]any)
		for _, id := range ids {
			if slices.Contains(fixture.TestIDs, id) {
				continue
			}
			tests, err := loader.load()
			if err != nil {
				return err
			}
			test, known := tests[id]
			if !known {
				continue
			}
			sources, ok := testSources(test, outputs, allowed)
			if !ok {
				continue
			}
			fixture.TestIDs = append(fixture.TestIDs, id)
			for _, source := range sources {
				fixture.PipelineOutputs[source] = OptionalizeExpressions(outputs[source])
			}
		}
	}
	return nil
}

// testSources returns the pipeline output sources a test reads, when every
// evidence binding resolves only through allowed steps.
func testSources(
	test dsl.TestDefinition,
	outputs map[string]any,
	allowed map[string]struct{},
) ([]string, bool) {
	sources := make([]string, 0, len(test.Evidence))
	for _, binding := range test.Evidence {
		source, key, found := strings.Cut(strings.TrimSpace(binding.From), outputsMarker+".")
		if !found {
			return nil, false
		}
		entry, _ := outputs[source].(map[string]any)
		output, _ := entry["output"].(map[string]any)
		value, exists := output[key]
		if !exists || !refsAllowed(value, allowed) {
			return nil, false
		}
		sources = append(sources, source)
	}
	return sources, true
}

func refsAllowed(value any, allowed map[string]struct{}) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if !refsAllowed(child, allowed) {
				return false
			}
		}
	case []any:
		for _, child := range typed {
			if !refsAllowed(child, allowed) {
				return false
			}
		}
	case string:
		for _, ref := range pipelineexpr.ExpressionRefs(typed) {
			stepID, _, _ := strings.Cut(ref, outputsMarker)
			if _, ok := allowed[stepID]; !ok {
				return false
			}
		}
	}
	return true
}

// StringList returns the non-empty strings of a decoded YAML or JSON list.
func StringList(value any) []string {
	items, _ := value.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && text != "" {
			out = append(out, text)
		}
	}
	return out
}
