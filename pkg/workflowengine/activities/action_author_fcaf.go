// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/inflector"
)

// Action author activities the Python agent calls to author FCAF actions.
const (
	ListActionAuthorFCAFRolesActivityName     = "List action author FCAF roles"
	DescribeActionAuthorFCAFRoleActivityName  = "Describe action author FCAF role"
	CheckActionAuthorFCAFEvidenceActivityName = "Check action author FCAF evidence"
)

const (
	// fcafDescribeFixtures caps the fixtures Describe returns.
	fcafDescribeFixtures = 5
	// fcafEvidenceScreenshots caps the screenshot names of an evidence check.
	fcafEvidenceScreenshots = 64
	// fcafEvidenceScreenshotURL prefixes the screenshot URLs an evidence
	// check makes up: the validators read only their file names.
	fcafEvidenceScreenshotURL = "https://credimi.invalid/fcaf-authoring/"
	// fcafEvidenceNoTests is the status of an evidence check whose fixture
	// runs no test.
	fcafEvidenceNoTests = "no_tests"
)

// fcafSessionID bounds the protocol session ID an evidence check fetches.
var fcafSessionID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// fcafSession reauthorizes the caller's session and returns its selection
// and FCAF scope, refusing a chat session.
func (l authorSessionLookup) fcafSession(
	ctx context.Context,
	base *workflowengine.BaseActivity,
	app core.App,
	caller authorCaller,
) (actionauthor.Selection, *actionauthor.FCAFScope, error) {
	selection, scope, err := l.sessionScope(ctx, base, app, caller)
	if err != nil {
		return actionauthor.Selection{}, nil, err
	}
	if scope == nil {
		return actionauthor.Selection{}, nil, actionAuthorError(base, fmt.Errorf(
			"%w: this session does not author FCAF actions", actionauthor.ErrForbidden,
		))
	}
	return selection, scope, nil
}

// ListActionAuthorFCAFRolesOutput is the session suite, the contract's
// authoring rules, the coverage of every role of the suite in the session
// wallet, the newest saved FCAF profile, null when none, and the FCAF
// findings about the wallet that profile implies (actionauthor.FCAFWarnings),
// which the wallet's coverage shows.
type ListActionAuthorFCAFRolesOutput struct {
	Suite    string                          `json:"suite"`
	Rules    []string                        `json:"rules"`
	Roles    []actionauthor.FCAFRoleCoverage `json:"roles"`
	Profile  map[string]any                  `json:"profile"`
	Warnings []string                        `json:"warnings"`
}

// ListActionAuthorFCAFRolesActivity lists the FCAF roles of the caller
// session's suite and the session wallet's action for each.
type ListActionAuthorFCAFRolesActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
}

// NewListActionAuthorFCAFRolesActivity binds the role listing to the application.
func NewListActionAuthorFCAFRolesActivity(app core.App) *ListActionAuthorFCAFRolesActivity {
	return &ListActionAuthorFCAFRolesActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: ListActionAuthorFCAFRolesActivityName,
		},
		authorSessionLookup: newAuthorSessionLookup(app),
		app:                 app,
	}
}

// Name returns the registered Temporal activity name.
func (a *ListActionAuthorFCAFRolesActivity) Name() string { return a.BaseActivity.Name }

// Execute lists the roles for the activity's actual caller.
func (a *ListActionAuthorFCAFRolesActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	if _, err := workflowengine.DecodePayload[struct{}](in.Payload); err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	output, err := a.list(ctx, activityCaller(ctx))
	if err != nil {
		return result, err
	}
	result.Output = output
	return result, nil
}

func (a *ListActionAuthorFCAFRolesActivity) list(
	ctx context.Context,
	caller authorCaller,
) (ListActionAuthorFCAFRolesOutput, error) {
	selection, scope, err := a.fcafSession(ctx, &a.BaseActivity, a.app, caller)
	if err != nil {
		return ListActionAuthorFCAFRolesOutput{}, err
	}
	roles, profile, err := actionauthor.FCAFCoverage(a.app, selection, scope)
	if err != nil {
		return ListActionAuthorFCAFRolesOutput{}, actionAuthorError(&a.BaseActivity, err)
	}
	return ListActionAuthorFCAFRolesOutput{
		Suite:    scope.Suite,
		Rules:    nonNil(scope.Contract.Rules),
		Roles:    nonNil(roles),
		Profile:  profile,
		Warnings: actionauthor.FCAFWarnings(profile),
	}, nil
}

// DescribeActionAuthorFCAFRoleInput names a role of the session suite.
type DescribeActionAuthorFCAFRoleInput struct {
	Role string `json:"role"`
}

// DescribeActionAuthorFCAFRoleOutput is a role of the session suite with its
// reference code, the roles its action starts from, the contract's
// authoring rules and Credimi-built plans: Plan for a role without
// parameters, one per fixture otherwise. PIDIssuance holds the parameter
// source of the injected PID issuance per PID issuer (walletroles.PIDIssuers),
// then per PID format ("sd-jwt", "mdoc").
// Plans carry no prerequisites: the agent adds the wallet's actions or
// candidates for StartsFrom, in order.
type DescribeActionAuthorFCAFRoleOutput struct {
	Role        walletroles.Role                              `json:"role"`
	RoleSHA256  string                                        `json:"role_sha256"`
	StartsFrom  []string                                      `json:"starts_from"`
	Rules       []string                                      `json:"rules"`
	Plan        *actionauthor.Spec                            `json:"plan"`
	Fixtures    []FCAFFixtureOption                           `json:"fixtures"`
	PIDIssuance map[string]map[string]actionauthor.SourceSpec `json:"pid_issuance"`
	// BrowserAppID is the browser package a role that uses the browser may
	// target, as Save allows it.
	BrowserAppID string `json:"browser_app_id"`
}

// FCAFFixtureOption is a fixture of an FCAF scenario that exercises the role,
// with the plan whose parameter it generates. PIDIssuer is set only for the
// injected issuance's fixtures.
type FCAFFixtureOption struct {
	ID           string            `json:"id"`
	Scenario     string            `json:"scenario"`
	MobileStepID string            `json:"mobile_step_id"`
	Standalone   bool              `json:"standalone"`
	PIDFormat    string            `json:"pid_format"`
	PIDIssuer    string            `json:"pid_issuer"`
	TestIDs      []string          `json:"test_ids"`
	Plan         actionauthor.Spec `json:"plan"`
}

// DescribeActionAuthorFCAFRoleActivity describes a role of the caller
// session's suite.
type DescribeActionAuthorFCAFRoleActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
}

// NewDescribeActionAuthorFCAFRoleActivity binds the role description to the application.
func NewDescribeActionAuthorFCAFRoleActivity(app core.App) *DescribeActionAuthorFCAFRoleActivity {
	return &DescribeActionAuthorFCAFRoleActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: DescribeActionAuthorFCAFRoleActivityName,
		},
		authorSessionLookup: newAuthorSessionLookup(app),
		app:                 app,
	}
}

// Name returns the registered Temporal activity name.
func (a *DescribeActionAuthorFCAFRoleActivity) Name() string { return a.BaseActivity.Name }

// Execute describes the role for the activity's actual caller.
func (a *DescribeActionAuthorFCAFRoleActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[DescribeActionAuthorFCAFRoleInput](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	output, err := a.describe(ctx, activityCaller(ctx), payload)
	if err != nil {
		return result, err
	}
	result.Output = output
	return result, nil
}

func (a *DescribeActionAuthorFCAFRoleActivity) describe(
	ctx context.Context,
	caller authorCaller,
	payload DescribeActionAuthorFCAFRoleInput,
) (DescribeActionAuthorFCAFRoleOutput, error) {
	_, scope, err := a.fcafSession(ctx, &a.BaseActivity, a.app, caller)
	if err != nil {
		return DescribeActionAuthorFCAFRoleOutput{}, err
	}
	output, err := describeFCAFRole(scope, strings.TrimSpace(payload.Role))
	if err != nil {
		return DescribeActionAuthorFCAFRoleOutput{}, actionAuthorError(&a.BaseActivity, err)
	}
	return output, nil
}

// describeFCAFRole describes the role name of the scope suite.
func describeFCAFRole(
	scope *actionauthor.FCAFScope,
	name string,
) (DescribeActionAuthorFCAFRoleOutput, error) {
	role, err := actionauthor.FCAFSuiteRole(scope, name)
	if err != nil {
		return DescribeActionAuthorFCAFRoleOutput{}, err
	}
	output := DescribeActionAuthorFCAFRoleOutput{
		Role:        role,
		RoleSHA256:  role.SHA256(),
		StartsFrom:  nonNil(scope.Contract.StartsFrom(role)),
		Rules:       nonNil(scope.Contract.Rules),
		Fixtures:    []FCAFFixtureOption{},
		PIDIssuance: map[string]map[string]actionauthor.SourceSpec{},
		// The browser package Save allows for roles that use the browser.
		BrowserAppID: scope.Contract.Reference.AppIDs[walletroles.AppBrowser],
	}
	// The PID issuances the generator may inject, per issuer and PID format.
	issuances, err := scope.Contract.Fixtures(
		fcafTemplatesRoot(),
		scope.Suite,
		scope.Contract.Injected(walletroles.InjectedIssuance).Name,
	)
	if err != nil {
		return DescribeActionAuthorFCAFRoleOutput{}, err
	}
	for _, issuance := range issuances {
		if output.PIDIssuance[issuance.PIDIssuer] == nil {
			output.PIDIssuance[issuance.PIDIssuer] = map[string]actionauthor.SourceSpec{}
		}
		output.PIDIssuance[issuance.PIDIssuer][issuance.PIDFormat] = actionauthor.SourceSpec{
			Kind:      actionauthor.SourceHTTP,
			Request:   httpSourceRequest(issuance.Request),
			ValuePath: issuance.ValuePath,
		}
	}
	if len(role.Parameters) == 0 {
		plan := fcafRolePlan(role)
		output.Plan = &plan
		return output, nil
	}
	fixtures, err := scope.Contract.Fixtures(fcafTemplatesRoot(), scope.Suite, role.Name)
	if err != nil {
		return DescribeActionAuthorFCAFRoleOutput{}, err
	}
	for _, fixture := range fixtures[:min(len(fixtures), fcafDescribeFixtures)] {
		output.Fixtures = append(output.Fixtures, FCAFFixtureOption{
			ID:           fixture.ID,
			Scenario:     fixture.Scenario,
			MobileStepID: fixture.MobileStepID,
			Standalone:   fixture.Standalone,
			PIDFormat:    fixture.PIDFormat,
			PIDIssuer:    fixture.PIDIssuer,
			TestIDs:      nonNil(fixture.TestIDs),
			Plan:         fixturePlan(role, fixture),
		})
	}
	return output, nil
}

// fcafRolePlan is the plan of the role's action without parameters or
// prerequisites.
func fcafRolePlan(role walletroles.Role) actionauthor.Spec {
	slot, _ := actionauthor.SlotForCategory(role.Category)
	return actionauthor.Spec{
		Slot:        slot,
		Outcome:     actionauthor.FCAFOutcome(role.Outcome),
		WaitSeconds: role.WaitSeconds,
		FCAFRole:    role.Name,
	}
}

// fixturePlan is the role's plan whose parameter the fixture's request
// generates, capturing the protocol session its fetches read.
func fixturePlan(role walletroles.Role, fixture walletroles.Fixture) actionauthor.Spec {
	plan := fcafRolePlan(role)
	if fixture.Request.Method == "" {
		return plan
	}
	source := actionauthor.SourceSpec{
		Kind:      actionauthor.SourceHTTP,
		Request:   httpSourceRequest(fixture.Request),
		ValuePath: fixture.ValuePath,
	}
	if len(fixture.Fetches) > 0 {
		source.Captures = map[string]string{"session_id": "session_id"}
	}
	plan.Parameters = []actionauthor.ParameterSpec{{Name: fixture.Parameter, Source: source}}
	return plan
}

func httpSourceRequest(request walletroles.FixtureRequest) *actionauthor.HTTPSourceRequest {
	return &actionauthor.HTTPSourceRequest{
		Method:  request.Method,
		URL:     request.URL,
		Headers: request.Headers,
		Body:    request.Body,
	}
}

// CheckActionAuthorFCAFEvidenceInput is a candidate's plan, the fixture its
// replay ran, the values captured when the fixture's parameter was generated
// and the names of the screenshots the replay stored.
type CheckActionAuthorFCAFEvidenceInput struct {
	Plan        actionauthor.Spec `json:"plan"`
	Fixture     string            `json:"fixture"`
	Captures    map[string]string `json:"captures"`
	Screenshots []string          `json:"screenshots"`
}

// CheckActionAuthorFCAFEvidenceOutput is the verdict of the fixture's FCAF
// tests on the replay: "no_tests" when the fixture runs none, else the
// report's status ("passed" or "failed").
type CheckActionAuthorFCAFEvidenceOutput struct {
	Status string                   `json:"status"`
	Tests  []FCAFEvidenceTestResult `json:"tests"`
}

// FCAFEvidenceTestResult is one test's verdict and its assertions that did
// not pass.
type FCAFEvidenceTestResult struct {
	ID       string                  `json:"id"`
	Status   string                  `json:"status"`
	Message  string                  `json:"message,omitempty"`
	Problems []FCAFEvidenceAssertion `json:"problems"`
}

// FCAFEvidenceAssertion is an assertion of a test.
type FCAFEvidenceAssertion struct {
	ID        string `json:"id"`
	Validator string `json:"validator"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// CheckActionAuthorFCAFEvidenceActivity runs the FCAF validators of a
// fixture on a candidate's replay: the protocol session the fixture's
// fetches read and the screenshots the replay stored. The screenshot names
// come from the agent: this is an authoring aid that the reviewing user
// confirms, not a security boundary.
type CheckActionAuthorFCAFEvidenceActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
	// transport sends the fetches.
	transport http.RoundTripper
}

// NewCheckActionAuthorFCAFEvidenceActivity binds the evidence check to the application.
func NewCheckActionAuthorFCAFEvidenceActivity(app core.App) *CheckActionAuthorFCAFEvidenceActivity {
	return &CheckActionAuthorFCAFEvidenceActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: CheckActionAuthorFCAFEvidenceActivityName,
		},
		authorSessionLookup: newAuthorSessionLookup(app),
		app:                 app,
		transport:           actionAuthorTransport(),
	}
}

// Name returns the registered Temporal activity name.
func (a *CheckActionAuthorFCAFEvidenceActivity) Name() string { return a.BaseActivity.Name }

// Execute checks the evidence for the activity's actual caller.
func (a *CheckActionAuthorFCAFEvidenceActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[CheckActionAuthorFCAFEvidenceInput](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	output, err := a.check(ctx, activityCaller(ctx), payload)
	if err != nil {
		return result, err
	}
	result.Output = output
	return result, nil
}

func (a *CheckActionAuthorFCAFEvidenceActivity) check(
	ctx context.Context,
	caller authorCaller,
	payload CheckActionAuthorFCAFEvidenceInput,
) (CheckActionAuthorFCAFEvidenceOutput, error) {
	selection, scope, err := a.fcafSession(ctx, &a.BaseActivity, a.app, caller)
	if err != nil {
		return CheckActionAuthorFCAFEvidenceOutput{}, err
	}
	output, err := a.checkEvidence(ctx, selection, scope, payload)
	if err != nil {
		return CheckActionAuthorFCAFEvidenceOutput{}, actionAuthorError(&a.BaseActivity, err)
	}
	return output, nil
}

func (a *CheckActionAuthorFCAFEvidenceActivity) checkEvidence(
	ctx context.Context,
	selection actionauthor.Selection,
	scope *actionauthor.FCAFScope,
	payload CheckActionAuthorFCAFEvidenceInput,
) (CheckActionAuthorFCAFEvidenceOutput, error) {
	plan, err := SessionPlan(a.app, selection, payload.Plan, scope)
	if err != nil {
		return CheckActionAuthorFCAFEvidenceOutput{}, err
	}
	if plan.FCAFRole == "" {
		return CheckActionAuthorFCAFEvidenceOutput{}, fmt.Errorf(
			"%w: the plan has no fcaf_role", actionauthor.ErrInvalid,
		)
	}
	fixture, err := fcafEvidenceFixture(scope, plan.FCAFRole, payload.Fixture)
	if err != nil {
		return CheckActionAuthorFCAFEvidenceOutput{}, err
	}
	if err := checkEvidenceInput(fixture, payload); err != nil {
		return CheckActionAuthorFCAFEvidenceOutput{}, err
	}
	if len(fixture.TestIDs) == 0 {
		return CheckActionAuthorFCAFEvidenceOutput{
			Status: fcafEvidenceNoTests,
			Tests:  []FCAFEvidenceTestResult{},
		}, nil
	}
	dataCtx, err := a.evidenceContext(ctx, fixture, payload)
	if err != nil {
		return CheckActionAuthorFCAFEvidenceOutput{}, err
	}
	resolvedValue, err := pipelineinternal.ResolveExpressions(fixture.PipelineOutputs, dataCtx)
	if err != nil {
		return CheckActionAuthorFCAFEvidenceOutput{}, fmt.Errorf(
			"%w: resolve the evidence: %w", actionauthor.ErrInvalid, err,
		)
	}
	resolved, _ := resolvedValue.(map[string]any)
	cat, err := catalog.Load(filepath.Join(fcafTemplatesRoot(), filepath.FromSlash(scope.Suite)))
	if err != nil {
		return CheckActionAuthorFCAFEvidenceOutput{}, fmt.Errorf("load the FCAF catalog: %w", err)
	}
	fcafEngine, err := engine.New(nil)
	if err != nil {
		return CheckActionAuthorFCAFEvidenceOutput{}, fmt.Errorf("create the FCAF engine: %w", err)
	}
	report, err := fcafEngine.ExecuteCatalog(
		ctx,
		cat,
		fixture.TestIDs,
		scope.Suite,
		nil,
		evidence.Bundle{PipelineOutputs: resolved},
	)
	if err != nil {
		return CheckActionAuthorFCAFEvidenceOutput{}, fmt.Errorf("run the FCAF tests: %w", err)
	}
	report.PopulateDerivedViews()
	return evidenceOutput(report), nil
}

// fcafEvidenceFixture returns the fixture id of the role.
func fcafEvidenceFixture(
	scope *actionauthor.FCAFScope,
	role, id string,
) (walletroles.Fixture, error) {
	fixtures, err := scope.Contract.Fixtures(fcafTemplatesRoot(), scope.Suite, role)
	if err != nil {
		return walletroles.Fixture{}, err
	}
	ids := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		if fixture.ID == id {
			return fixture, nil
		}
		ids = append(ids, fixture.ID)
	}
	return walletroles.Fixture{}, fmt.Errorf(
		"%w: fixture %s is not one of %s", actionauthor.ErrInvalid, id, strings.Join(ids, ", "),
	)
}

// checkEvidenceInput checks the screenshot names and the captures the
// fixture's fetches need.
func checkEvidenceInput(
	fixture walletroles.Fixture,
	payload CheckActionAuthorFCAFEvidenceInput,
) error {
	if len(payload.Screenshots) > fcafEvidenceScreenshots {
		return fmt.Errorf(
			"%w: screenshots: at most %d names", actionauthor.ErrInvalid, fcafEvidenceScreenshots,
		)
	}
	for _, name := range payload.Screenshots {
		if !actionauthor.ValidScreenshotName(name) {
			return fmt.Errorf(
				"%w: screenshots: %q is not a screenshot name "+
					"(1-64 letters, digits, '-' or '_')",
				actionauthor.ErrInvalid, name,
			)
		}
	}
	if len(fixture.Fetches) == 0 {
		return nil
	}
	sessionID := payload.Captures["session_id"]
	if sessionID == "" {
		return fmt.Errorf("%w: captures.session_id is required", actionauthor.ErrInvalid)
	}
	// The session ID goes into the fetch URLs' path as is.
	if !fcafSessionID.MatchString(sessionID) {
		return fmt.Errorf(
			"%w: captures.session_id must be 1-128 letters, digits, '-' or '_'",
			actionauthor.ErrInvalid,
		)
	}
	return nil
}

// evidenceContext is the data context the fixture's pipeline outputs resolve
// against: the outputs of its fetches of the protocol session, and the
// mobile step's screenshot URLs, named as Credimi stores them.
func (a *CheckActionAuthorFCAFEvidenceActivity) evidenceContext(
	ctx context.Context,
	fixture walletroles.Fixture,
	payload CheckActionAuthorFCAFEvidenceInput,
) (map[string]any, error) {
	dataCtx := make(map[string]any, len(fixture.Fetches)+1)
	for _, fetch := range fixture.Fetches {
		result, err := executeHTTPRequest(ctx, HTTPActivityPayload{
			Method: http.MethodGet,
			URL: strings.ReplaceAll(
				fetch.URL, "${session_id}", payload.Captures["session_id"],
			),
			Headers: map[string]string{"Accept": "application/json"},
		}, nil, &a.BaseActivity, sourceRedirectPolicy{next: a.transport}, actionAuthorStatusBodyLimit)
		if err != nil {
			return nil, fmt.Errorf("%w: fetch %s: %w", actionauthor.ErrInvalid, fetch.StepID, err)
		}
		response, _ := result.Output.(map[string]any)
		if status, _ := response["status"].(int); status < 200 || status > 299 {
			return nil, fmt.Errorf(
				"%w: fetch %s answered HTTP %v",
				actionauthor.ErrInvalid,
				fetch.StepID,
				response["status"],
			)
		}
		dataCtx[fetch.StepID] = map[string]any{"outputs": result.Output}
	}
	urls := make([]any, 0, len(payload.Screenshots))
	for _, name := range payload.Screenshots {
		urls = append(urls, fcafEvidenceScreenshotURL+
			inflector.Snakecase(fixture.MobileStepID+"-"+name)+"_0000000000.png")
	}
	dataCtx[fixture.MobileStepID] = map[string]any{"outputs": map[string]any{
		"flow_output": map[string]any{"maestro_screenshot_urls": urls},
	}}
	return dataCtx, nil
}

// evidenceOutput is the verdict of every test of report and its assertions
// that did not pass.
func evidenceOutput(report engine.Report) CheckActionAuthorFCAFEvidenceOutput {
	output := CheckActionAuthorFCAFEvidenceOutput{
		Status: report.Status,
		Tests:  make([]FCAFEvidenceTestResult, 0, len(report.Tests)),
	}
	for _, test := range report.Tests {
		result := FCAFEvidenceTestResult{
			ID:       test.ID,
			Status:   string(test.Status),
			Message:  test.Message,
			Problems: []FCAFEvidenceAssertion{},
		}
		for _, assertion := range test.Assertions {
			if assertion.Status == validators.StatusPass {
				continue
			}
			result.Problems = append(result.Problems, FCAFEvidenceAssertion{
				ID:        assertion.ID,
				Validator: assertion.Validator,
				Status:    string(assertion.Status),
				Message:   assertion.Message,
			})
		}
		output.Tests = append(output.Tests, result)
	}
	return output
}

// nonNil returns values, or an empty slice when it is nil.
func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
