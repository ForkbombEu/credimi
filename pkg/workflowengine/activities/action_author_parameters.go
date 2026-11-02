// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/safehttp"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/sdk/activity"
)

// Action author activities the Python agent calls on the organization worker.
const (
	GenerateActionAuthorParametersActivityName   = "Generate action author parameter values"
	CheckActionAuthorProtocolOutcomeActivityName = "Check action author protocol outcome"
)

// Parameter scopes of GenerateActionAuthorParametersInput.
const (
	ParameterScopeCandidate    = "candidate"
	ParameterScopePrerequisite = "prerequisite"
)

const (
	actionAuthorHTTPTimeout     = 30 * time.Second
	actionAuthorStatusInterval  = 2 * time.Second
	actionAuthorStatusTimeout   = 30 * time.Second
	actionAuthorStatusBodyLimit = 1 << 20
	actionAuthorSourceBodyLimit = actionauthor.SourceBodyLimit
	actionAuthorSourceRedirects = 5
)

var statusCaptureReference = regexp.MustCompile(`\$\{([^}]*)\}`)

// actionAuthorTransport refuses private and loopback destinations.
func actionAuthorTransport() http.RoundTripper {
	return safehttp.NewClient(safehttp.Config{Timeout: actionAuthorHTTPTimeout}).Transport
}

// sourceRedirectPolicy is the redirect policy of http source requests: at
// most actionAuthorSourceRedirects redirects, never from https to http. It
// checks each request as a transport because executeHTTPRequest builds its own
// client; a redirected request carries the response that caused it.
type sourceRedirectPolicy struct {
	next http.RoundTripper
}

func (p sourceRedirectPolicy) RoundTrip(req *http.Request) (*http.Response, error) {
	redirects := 0
	for previous := req.Response; previous != nil; previous = previous.Request.Response {
		redirects++
		if redirects == 1 && previous.Request.URL.Scheme == "https" && req.URL.Scheme != "https" {
			return nil, fmt.Errorf("refused redirect from https to %s", req.URL.Scheme)
		}
	}
	if redirects > actionAuthorSourceRedirects {
		return nil, fmt.Errorf("stopped after %d redirects", actionAuthorSourceRedirects)
	}
	next := p.next
	if next == nil {
		next = http.DefaultTransport
	}
	return next.RoundTrip(req)
}

// GenerateActionAuthorParametersInput selects the parameter list of the
// caller's session: the candidate's, or the prerequisite's at Index.
// PlanToken is the confirmed plan's token when the session has no start spec.
type GenerateActionAuthorParametersInput struct {
	Scope     string `json:"scope"`
	Index     int    `json:"index"`
	PlanToken string `json:"plan_token,omitempty"`
}

// GenerateActionAuthorParametersOutput carries fresh parameter values and, for
// http sources, the captured response values by parameter.
type GenerateActionAuthorParametersOutput struct {
	Values   map[string]string            `json:"values"`
	Captures map[string]map[string]string `json:"captures"`
}

// GenerateActionAuthorParametersActivity generates fresh values for the
// parameters the caller's session declared.
type GenerateActionAuthorParametersActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
	// transport sends the http sources' requests.
	transport http.RoundTripper
	// runStepCI generates the deeplink of dynamic records.
	runStepCI stepCIDeeplinkFunc
}

// NewGenerateActionAuthorParametersActivity binds parameter generation to the application.
func NewGenerateActionAuthorParametersActivity(
	app core.App,
) *GenerateActionAuthorParametersActivity {
	return &GenerateActionAuthorParametersActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: GenerateActionAuthorParametersActivityName,
		},
		authorSessionLookup: newAuthorSessionLookup(),
		app:                 app,
		transport:           actionAuthorTransport(),
		runStepCI:           runStepCIDeeplink,
	}
}

// Name returns the registered Temporal activity name.
func (a *GenerateActionAuthorParametersActivity) Name() string { return a.BaseActivity.Name }

// Execute generates the values for the activity's actual caller.
func (a *GenerateActionAuthorParametersActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[GenerateActionAuthorParametersInput](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	output, err := a.generate(ctx, activityCaller(ctx), payload)
	if err != nil {
		return result, err
	}
	result.Output = output
	return result, nil
}

func (a *GenerateActionAuthorParametersActivity) generate(
	ctx context.Context,
	caller authorCaller,
	payload GenerateActionAuthorParametersInput,
) (GenerateActionAuthorParametersOutput, error) {
	output := GenerateActionAuthorParametersOutput{
		Values:   map[string]string{},
		Captures: map[string]map[string]string{},
	}
	ref, err := a.callerSession(ctx, &a.BaseActivity, caller)
	if err != nil {
		return output, err
	}
	plan, err := EffectivePlan(a.app, ref, caller.WorkflowID, payload.PlanToken)
	if err != nil {
		return output, actionAuthorError(&a.BaseActivity, err)
	}
	selection, err := actionauthor.Select(a.app, caller.Namespace, ref.Session.BuildRequest)
	if err != nil {
		return output, actionAuthorError(&a.BaseActivity, err)
	}
	parameters, err := scopedParameters(plan, payload)
	if err != nil {
		return output, actionAuthorError(&a.BaseActivity, err)
	}
	for _, parameter := range parameters {
		value, captures, err := a.value(ctx, selection.Organization.Id, parameter)
		if err != nil {
			return output, actionAuthorError(&a.BaseActivity, err)
		}
		output.Values[parameter.Name] = value
		if len(captures) > 0 {
			output.Captures[parameter.Name] = captures
		}
	}
	return output, nil
}

// scopedParameters returns the candidate's parameters or those of one prerequisite.
func scopedParameters(
	spec actionauthor.Spec,
	payload GenerateActionAuthorParametersInput,
) ([]actionauthor.ParameterSpec, error) {
	switch payload.Scope {
	case ParameterScopeCandidate:
		return spec.Parameters, nil
	case ParameterScopePrerequisite:
		if payload.Index < 0 || payload.Index >= len(spec.Prerequisites) {
			return nil, fmt.Errorf(
				"%w: the session has no prerequisite %d",
				actionauthor.ErrInvalid,
				payload.Index,
			)
		}
		return spec.Prerequisites[payload.Index].Parameters, nil
	default:
		return nil, fmt.Errorf(
			"%w: scope must be candidate or prerequisite",
			actionauthor.ErrInvalid,
		)
	}
}

// value generates one parameter value. Generation failures are invalid
// parameters, reported to the agent with the parameter's name.
func (a *GenerateActionAuthorParametersActivity) value(
	ctx context.Context,
	orgID string,
	parameter actionauthor.ParameterSpec,
) (string, map[string]string, error) {
	source := parameter.Source
	invalid := func(reason any) error {
		return fmt.Errorf("%w: parameter %s: %v", actionauthor.ErrInvalid, parameter.Name, reason)
	}
	switch source.Kind {
	case actionauthor.SourceLiteral:
		return source.Value, nil, nil
	case actionauthor.SourceCredentialOffer:
		record, err := actionauthor.CheckRecordAccess(
			a.app,
			"credentials",
			source.Identifier,
			orgID,
		)
		if err != nil {
			return "", nil, err
		}
		offer, secrets, err := recordCredentialOffer(a.app, record)
		if err != nil {
			return "", nil, invalid(err)
		}
		if !offer.Dynamic {
			return offer.CredentialOffer, nil, nil
		}
		deeplink, err := a.runStepCI(ctx, offer.Code, secrets)
		if err != nil {
			return "", nil, invalid(err)
		}
		return deeplink, nil, nil
	case actionauthor.SourceUseCaseVerification:
		record, err := actionauthor.CheckRecordAccess(
			a.app,
			"use_cases_verifications",
			source.Identifier,
			orgID,
		)
		if err != nil {
			return "", nil, err
		}
		secrets, err := parseRecordSecrets(record)
		if err != nil {
			return "", nil, invalid(err)
		}
		deeplink, err := a.runStepCI(ctx, record.GetString("yaml"), secrets)
		if err != nil {
			return "", nil, invalid(err)
		}
		return deeplink, nil, nil
	case actionauthor.SourceHTTP:
		return a.httpValue(ctx, source, invalid)
	default:
		return "", nil, invalid("unknown source kind " + source.Kind)
	}
}

func (a *GenerateActionAuthorParametersActivity) httpValue(
	ctx context.Context,
	source actionauthor.SourceSpec,
	invalid func(any) error,
) (string, map[string]string, error) {
	if source.Request == nil {
		return "", nil, invalid("the http source has no request")
	}
	result, err := executeHTTPRequest(ctx, HTTPActivityPayload{
		Method:  source.Request.Method,
		URL:     source.Request.URL,
		Headers: source.Request.Headers,
		Body:    source.Request.Body,
	}, nil, &a.BaseActivity, sourceRedirectPolicy{next: a.transport}, actionAuthorSourceBodyLimit)
	if err != nil {
		return "", nil, invalid(err)
	}
	response, _ := result.Output.(map[string]any)
	if status, _ := response["status"].(int); status < 200 || status > 299 {
		return "", nil, invalid(fmt.Sprintf("%s %s answered HTTP %v",
			source.Request.Method, source.Request.URL, response["status"]))
	}
	body := response["body"]
	value, ok := actionauthor.LookupPath(body, source.ValuePath)
	if !ok {
		return "", nil, invalid(source.ValuePath + " is not in the response")
	}
	captures := make(map[string]string, len(source.Captures))
	for name, path := range source.Captures {
		captured, ok := actionauthor.LookupPath(body, path)
		if !ok {
			return "", nil, invalid(
				fmt.Sprintf("capture %s: %s is not in the response", name, path),
			)
		}
		captures[name] = captured
	}
	return value, captures, nil
}

// CheckActionAuthorProtocolOutcomeInput names a candidate parameter with a
// status check and the values captured when its value was generated;
// PlanToken is the confirmed plan's token when the session has no start spec.
type CheckActionAuthorProtocolOutcomeInput struct {
	Parameter string            `json:"parameter"`
	Captures  map[string]string `json:"captures"`
	PlanToken string            `json:"plan_token,omitempty"`
}

// CheckActionAuthorProtocolOutcomeOutput reports whether the protocol session
// reached a status the session's outcome expects.
type CheckActionAuthorProtocolOutcomeOutput struct {
	Reached  bool     `json:"reached"`
	Status   string   `json:"status"`
	Expected []string `json:"expected"`
}

// CheckActionAuthorProtocolOutcomeActivity polls the issuer or verifier
// session status declared by a candidate parameter.
type CheckActionAuthorProtocolOutcomeActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
	// transport sends the status requests.
	transport http.RoundTripper
	interval  time.Duration
	timeout   time.Duration
}

// NewCheckActionAuthorProtocolOutcomeActivity returns the protocol outcome check.
func NewCheckActionAuthorProtocolOutcomeActivity(
	app core.App,
) *CheckActionAuthorProtocolOutcomeActivity {
	return &CheckActionAuthorProtocolOutcomeActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: CheckActionAuthorProtocolOutcomeActivityName,
		},
		authorSessionLookup: newAuthorSessionLookup(),
		app:                 app,
		transport:           actionAuthorTransport(),
		interval:            actionAuthorStatusInterval,
		timeout:             actionAuthorStatusTimeout,
	}
}

// Name returns the registered Temporal activity name.
func (a *CheckActionAuthorProtocolOutcomeActivity) Name() string { return a.BaseActivity.Name }

// Execute checks the outcome for the activity's actual caller.
func (a *CheckActionAuthorProtocolOutcomeActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[CheckActionAuthorProtocolOutcomeInput](in.Payload)
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

func (a *CheckActionAuthorProtocolOutcomeActivity) check(
	ctx context.Context,
	caller authorCaller,
	payload CheckActionAuthorProtocolOutcomeInput,
) (CheckActionAuthorProtocolOutcomeOutput, error) {
	var output CheckActionAuthorProtocolOutcomeOutput
	ref, err := a.callerSession(ctx, &a.BaseActivity, caller)
	if err != nil {
		return output, err
	}
	plan, err := EffectivePlan(a.app, ref, caller.WorkflowID, payload.PlanToken)
	if err != nil {
		return output, actionAuthorError(&a.BaseActivity, err)
	}
	status, err := candidateStatus(plan, payload.Parameter)
	if err != nil {
		return output, actionAuthorError(&a.BaseActivity, err)
	}
	statusURL, err := statusURL(status.URL, payload.Captures)
	if err != nil {
		return output, actionAuthorError(&a.BaseActivity, err)
	}
	expected, opposite := status.Success, status.Rejection
	if plan.Outcome == actionauthor.OutcomeRejection {
		expected, opposite = status.Rejection, status.Success
	}
	output.Expected = expected
	output.Reached, output.Status = a.poll(ctx, statusURL, status.Path, expected, opposite)
	return output, nil
}

// candidateStatus returns the status check of the session's own candidate
// parameter, never one supplied by the caller.
func candidateStatus(spec actionauthor.Spec, name string) (*actionauthor.StatusSpec, error) {
	for _, parameter := range spec.Parameters {
		if parameter.Name == name && parameter.Source.Status != nil {
			return parameter.Source.Status, nil
		}
	}
	return nil, fmt.Errorf(
		"%w: parameter %s has no protocol status check",
		actionauthor.ErrInvalid,
		name,
	)
}

// statusURL fills the ${capture} references of a status URL. The filled URL
// must keep the template's host, so no capture can redirect the poll.
func statusURL(template string, captures map[string]string) (string, error) {
	var missing string
	filled := statusCaptureReference.ReplaceAllStringFunc(template, func(reference string) string {
		name := statusCaptureReference.FindStringSubmatch(reference)[1]
		value, ok := captures[name]
		if !ok && missing == "" {
			missing = name
		}
		return url.PathEscape(value)
	})
	if missing != "" {
		return "", fmt.Errorf("%w: capture %s is missing", actionauthor.ErrInvalid, missing)
	}
	host, err := actionauthor.StatusTemplateHost(template)
	if err != nil {
		return "", fmt.Errorf("%w: status url: %w", actionauthor.ErrInvalid, err)
	}
	parsed, err := url.Parse(filled)
	if err != nil || parsed.Scheme != "https" || parsed.Host != host {
		return "", fmt.Errorf(
			"%w: the filled status url must keep the host %s",
			actionauthor.ErrInvalid,
			host,
		)
	}
	return filled, nil
}

// poll reads the status until it is in expected or opposite, or the timeout
// passes; it then reports the last status read, or why none could be read.
func (a *CheckActionAuthorProtocolOutcomeActivity) poll(
	ctx context.Context,
	statusURL, path string,
	expected, opposite []string,
) (bool, string) {
	client := &http.Client{Timeout: actionAuthorHTTPTimeout, Transport: a.transport}
	deadline := time.Now().Add(a.timeout)
	last, lastErr := "", ""
	for attempt := 1; ctx.Err() == nil; attempt++ {
		if activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx, attempt)
		}
		status, err := readStatus(ctx, client, statusURL, path)
		switch {
		case err != nil:
			lastErr = err.Error()
		case slices.Contains(expected, status):
			return true, status
		case slices.Contains(opposite, status):
			return false, status
		default:
			last = status
		}
		if !time.Now().Add(a.interval).Before(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			lastErr = ctx.Err().Error()
		case <-time.After(a.interval):
		}
	}
	if last == "" {
		return false, "unreachable: " + lastErr
	}
	return false, last
}

func readStatus(ctx context.Context, client *http.Client, statusURL, path string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, actionAuthorStatusBodyLimit))
	if err != nil {
		return "", err
	}
	var body any
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", fmt.Errorf("HTTP %d: the response is not JSON", response.StatusCode)
	}
	status, ok := actionauthor.LookupPath(body, path)
	if !ok {
		return "", fmt.Errorf("HTTP %d: %s is not in the response", response.StatusCode, path)
	}
	return status, nil
}
