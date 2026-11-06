// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/pocketbase/pocketbase/core"
)

// Step uses and `with` keys of the pipeline steps a failed run's plan is
// suggested from, as in the workflow payload structs
// (workflows.MobileAutomationWorkflowPipelinePayload,
// workflows.GetCredentialOfferWorkflowPayload,
// workflows.GetUseCaseVerificationDeeplinkWorkflowPayload,
// activities.HTTPActivityPayload), which this package cannot import.
const (
	StepUseMobileAutomation     = "mobile-automation"
	stepUseCredentialOffer      = "credential-offer"
	stepUseVerificationDeeplink = "use-case-verification-deeplink"
	stepUseHTTPRequest          = "http-request"
	mobileActionIDKey           = "action_id"
	mobileActionsKey            = "actions"
	mobileParametersKey         = "parameters"
	mobileVersionIDKey          = "version_id"
	credentialIDKey             = "credential_id"
	useCaseIDKey                = "use_case_id"
)

// ExternalSourceVersion is the version_id of a pipeline step whose wallet
// was installed outside Credimi, as from the store.
const ExternalSourceVersion = "installed_from_external_source"

const (
	pipelineWorkflowIDPrefix = "Pipeline-"
	runsPathMarker           = "/my/tests/runs/"
	maestroTailLines         = 40
	maestroTailChars         = 4000
	httpBodyOutputMarker     = ".outputs.body."
	notOwnedNote             = "this action belongs to another organization: save a fixed copy under a new name, then change the pipeline to use it"
	otherWalletNote          = "this action is not of the session's wallet: start a session on %s"
	versionNote              = "the run used version %s, the session uses %s"
	parameterSourceNote      = "parameter %s comes from step %s (%s): describe its source"
	missingActionNote        = "action %s no longer exists"
	unknownCategoryNote      = "action category %s has no slot: the plan uses other"
)

var (
	// pipelinePathMarkers are the webapp paths of a pipeline.
	pipelinePathMarkers = []string{"/my/pipelines/", "/hub/pipelines/"}
	// maestroFailedLine is the agent's maestro.py _RESULT_LINE with status FAILED.
	maestroFailedLine = regexp.MustCompile(`^\s*(.+?)\.\.\. FAILED\s*$`)
	// maestroNoise matches the JVM warnings Maestro prints on every start.
	maestroNoise = regexp.MustCompile(`^(WARNING:|kotlin-logging:|SLF4J)`)
	// stepOutputReference is a parameter value that is exactly one step's outputs.
	stepOutputReference = regexp.MustCompile(
		`^\$\{\{\s*([A-Za-z0-9_-]+)\.outputs(?:\.[^}]*)?\s*\}\}$`,
	)
	// expressionReference finds the first name an expression refers to.
	expressionReference = regexp.MustCompile(`\$\{\{\s*([A-Za-z0-9_-]+)`)
)

// PipelineRunRef names a pipeline run (WorkflowID, with RunID empty for its
// latest run) or a pipeline (Pipeline) whose latest failed run is meant.
type PipelineRunRef struct {
	WorkflowID string
	RunID      string
	Pipeline   string
}

// ParsePipelineRunReference reads a run URL (/my/tests/runs/<workflow>[/<run>]),
// a pipeline URL (/my/pipelines/<org>/<pipeline>, /hub/pipelines/<org>/<pipeline>),
// a bare pipeline workflow ID or a bare org/pipeline path. Only the path
// of a URL counts.
func ParsePipelineRunReference(text string) (PipelineRunRef, error) {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)
	switch {
	case strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://"):
		parsed, err := url.Parse(text)
		if err == nil {
			if ref, ok := parseRunPath(parsed.Path); ok {
				return ref, nil
			}
		}
	case strings.HasPrefix(text, pipelineWorkflowIDPrefix) && !strings.ContainsAny(text, " \t\r\n/"):
		return PipelineRunRef{WorkflowID: text}, nil
	default:
		if path, ok := pipelinePath(text); ok {
			return PipelineRunRef{Pipeline: path}, nil
		}
	}
	return PipelineRunRef{}, invalidf("not a pipeline run or pipeline: %s", text)
}

// parseRunPath reads the run or pipeline a webapp URL path names.
func parseRunPath(path string) (PipelineRunRef, bool) {
	if _, rest, ok := strings.Cut(path, runsPathMarker); ok {
		segments := strings.Split(strings.TrimSuffix(rest, "/"), "/")
		if len(segments) > 2 || slices.Contains(segments, "") {
			return PipelineRunRef{}, false
		}
		ref := PipelineRunRef{WorkflowID: segments[0]}
		if len(segments) == 2 {
			ref.RunID = segments[1]
		}
		return ref, true
	}
	for _, marker := range pipelinePathMarkers {
		if _, rest, ok := strings.Cut(path, marker); ok {
			if pipeline, ok := pipelinePath(strings.TrimSuffix(rest, "/")); ok {
				return PipelineRunRef{Pipeline: pipeline}, true
			}
			return PipelineRunRef{}, false
		}
	}
	return PipelineRunRef{}, false
}

// pipelinePath accepts org/pipeline of two canonified segments.
func pipelinePath(text string) (string, bool) {
	segments := strings.Split(text, "/")
	if len(segments) != 2 {
		return "", false
	}
	for _, segment := range segments {
		if segment == "" || canonify.CanonifyPlain(segment) != segment {
			return "", false
		}
	}
	return text, true
}

// MaestroFailure returns the first command Maestro reports as FAILED in
// stdout (empty without one) and the last 40 non-empty lines that are not
// JVM noise, at most 4000 characters, keeping the end.
func MaestroFailure(stdout string) (string, string) {
	failedCommand := ""
	kept := []string{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimRight(line, "\r")
		if failedCommand == "" {
			if match := maestroFailedLine.FindStringSubmatch(line); match != nil {
				failedCommand = strings.TrimSpace(match[1])
			}
		}
		if strings.TrimSpace(line) == "" || maestroNoise.MatchString(line) {
			continue
		}
		kept = append(kept, line)
	}
	if len(kept) > maestroTailLines {
		kept = kept[len(kept)-maestroTailLines:]
	}
	tail := []rune(strings.Join(kept, "\n"))
	if len(tail) > maestroTailChars {
		tail = tail[len(tail)-maestroTailChars:]
	}
	return failedCommand, string(tail)
}

// SlotForCategory returns the slot whose actions are saved with category.
func SlotForCategory(category string) (string, bool) {
	for slot, rule := range Slots {
		if rule.Category == category {
			return slot, true
		}
	}
	return "", false
}

// PipelineRunFailure is a failed mobile-automation step of a pipeline run:
// the action it ran, why it failed and a suggested plan to fix it.
type PipelineRunFailure struct {
	StepID          string   `json:"step_id"`
	Action          string   `json:"action"`
	ActionName      string   `json:"action_name"`
	Candidates      []string `json:"candidates"`
	Owned           bool     `json:"owned"`
	InSessionWallet bool     `json:"in_session_wallet"`
	Version         string   `json:"version"`
	Code            string   `json:"code"`
	Message         string   `json:"message"`
	FailedCommand   string   `json:"failed_command"`
	OutputTail      string   `json:"output_tail"`
	Plan            *Spec    `json:"plan"`
	Notes           []string `json:"notes"`
}

// StepFailure is the failure a pipeline recorded for one step.
type StepFailure struct {
	Code    string
	Message string
	Stdout  string
}

// DescribeFailedStep describes the failed mobile-automation step
// definition.Steps[stepIndex] for the session's selection: the stored action
// it ran (accessible to the session's organization), its owner and wallet,
// the version it ran and, for one stored action, a suggested plan.
func DescribeFailedStep(
	app core.App,
	selection Selection,
	definition *pipeline.WorkflowDefinition,
	stepIndex int,
	failure StepFailure,
) (PipelineRunFailure, error) {
	step := definition.Steps[stepIndex]
	described := PipelineRunFailure{
		StepID:     step.ID,
		Candidates: []string{},
		Version:    stringValue(step.With.Payload[mobileVersionIDKey]),
		Code:       failure.Code,
		Message:    failure.Message,
		Notes:      []string{},
	}
	described.FailedCommand, described.OutputTail = MaestroFailure(failure.Stdout)
	if entries, ok := step.With.Payload[mobileActionsKey].([]any); ok {
		for _, entry := range entries {
			fields, _ := entry.(map[string]any)
			if id := canonify.NormalizePath(stringValue(fields["id"])); id != "" {
				described.Candidates = append(described.Candidates, id)
			}
		}
		return described, nil
	}
	action := canonify.NormalizePath(stringValue(step.With.Payload[mobileActionIDKey]))
	if action == "" {
		return described, nil
	}
	described.Action = action
	record, err := CheckRecordAccess(app, "wallet_actions", action, selection.Organization.Id)
	if isInvalid(err) {
		described.Notes = append(described.Notes, fmt.Sprintf(missingActionNote, action))
		return described, nil
	}
	if err != nil {
		return PipelineRunFailure{}, err
	}
	described.ActionName = record.GetString("name")
	described.Owned = record.GetString("owner") == selection.Organization.Id
	described.InSessionWallet = record.GetString("wallet") == selection.Wallet.Id
	if !described.Owned {
		described.Notes = append(described.Notes, notOwnedNote)
	}
	if !described.InSessionWallet {
		wallet, err := walletPath(app, record.GetString("wallet"))
		if err != nil {
			return PipelineRunFailure{}, err
		}
		described.Notes = append(described.Notes, fmt.Sprintf(otherWalletNote, wallet))
	}
	if described.Version != "" && !sameBuild(described.Version, selection) {
		described.Notes = append(described.Notes,
			fmt.Sprintf(versionNote, described.Version, selection.Identifier))
	}
	plan, notes, err := SuggestPlan(app, definition, stepIndex, record, selection)
	if err != nil {
		return PipelineRunFailure{}, err
	}
	described.Plan = &plan
	described.Notes = append(described.Notes, notes...)
	return described, nil
}

// sameBuild reports whether a step's version_id is the session's build; a
// store session's wallet is the one installed from an external source.
func sameBuild(version string, selection Selection) bool {
	version = canonify.NormalizePath(version)
	return version == selection.Identifier ||
		(selection.Source == SourceStore && version == ExternalSourceVersion)
}

func walletPath(app core.App, walletID string) (string, error) {
	wallet, err := app.FindRecordById("wallets", walletID)
	if err != nil {
		return "", fmt.Errorf("find wallet %s: %w", walletID, err)
	}
	path, err := canonify.BuildPath(
		app,
		wallet,
		canonify.CanonifyPaths["wallets"],
		wallet.GetString("canonified_name"),
	)
	if err != nil {
		return "", fmt.Errorf("build wallet path: %w", err)
	}
	return path, nil
}

// SuggestPlan suggests the plan that fixes action, the stored action that
// definition.Steps[stepIndex] runs: the slot of its category, a success
// outcome, the action itself as replaced when the selection's organization
// owns it, the earlier steps' actions of its wallet as prerequisites and the
// step's parameters as sources. The notes say what could not be translated.
// The plan is not checked: describe_action checks it.
func SuggestPlan(
	app core.App,
	definition *pipeline.WorkflowDefinition,
	stepIndex int,
	action *core.Record,
	selection Selection,
) (Spec, []string, error) {
	step := definition.Steps[stepIndex]
	notes := []string{}
	slot, ok := SlotForCategory(action.GetString("category"))
	if !ok {
		slot = "other"
		notes = append(notes, fmt.Sprintf(unknownCategoryNote, action.GetString("category")))
	}
	plan := Spec{Slot: slot, Outcome: OutcomeSuccess}
	if action.GetString("owner") == selection.Organization.Id {
		plan.Replaces = &ReplaceSpec{
			Action: canonify.NormalizePath(stringValue(step.With.Payload[mobileActionIDKey])),
		}
	}
	if Slots[slot].Prerequisites != PrereqForbidden {
		prerequisites, prerequisiteNotes, err := suggestPrerequisites(
			app,
			definition,
			stepIndex,
			action.GetString("wallet"),
			selection.Organization.Id,
		)
		if err != nil {
			return Spec{}, nil, err
		}
		plan.Prerequisites = prerequisites
		notes = append(notes, prerequisiteNotes...)
	}
	parameters, parameterNotes := suggestParameters(definition, step.StepSpec)
	plan.Parameters = parameters
	notes = append(notes, parameterNotes...)
	return plan, notes, nil
}

// suggestPrerequisites returns, in order, the actions of the earlier
// mobile-automation steps that run one stored action of walletID other than
// an install-app action, with their steps' parameters.
func suggestPrerequisites(
	app core.App,
	definition *pipeline.WorkflowDefinition,
	stepIndex int,
	walletID, orgID string,
) ([]PrerequisiteSpec, []string, error) {
	prerequisites := []PrerequisiteSpec{}
	notes := []string{}
	for _, step := range definition.Steps[:stepIndex] {
		if step.Use != StepUseMobileAutomation {
			continue
		}
		if _, many := step.With.Payload[mobileActionsKey]; many {
			continue
		}
		path := canonify.NormalizePath(stringValue(step.With.Payload[mobileActionIDKey]))
		if path == "" {
			continue
		}
		record, err := CheckRecordAccess(app, "wallet_actions", path, orgID)
		if isInvalid(err) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		if record.GetString("wallet") != walletID ||
			record.GetString("category") == SlotInstallApp {
			continue
		}
		parameters, parameterNotes := suggestParameters(definition, step.StepSpec)
		prerequisites = append(prerequisites, PrerequisiteSpec{
			Action:     path,
			Parameters: parameters,
		})
		notes = append(notes, parameterNotes...)
	}
	return prerequisites, notes, nil
}

// suggestParameters translates a mobile-automation step's parameters, by
// name, into sources: literals, and the outputs of credential-offer,
// use-case-verification-deeplink and http-request steps. Any other value is
// a note and is left out.
func suggestParameters(
	definition *pipeline.WorkflowDefinition,
	step pipeline.StepSpec,
) ([]ParameterSpec, []string) {
	values, _ := step.With.Payload[mobileParametersKey].(map[string]any)
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	parameters := []ParameterSpec{}
	notes := []string{}
	for _, name := range names {
		value := fmt.Sprint(values[name])
		source, ok := translateParameter(definition, value)
		if ok {
			parameters = append(parameters, ParameterSpec{Name: name, Source: source})
			continue
		}
		if !strings.Contains(value, "${{") {
			parameters = append(parameters, ParameterSpec{
				Name:   name,
				Source: SourceSpec{Kind: SourceLiteral, Value: value},
			})
			continue
		}
		stepID, use := "", "unknown"
		if match := expressionReference.FindStringSubmatch(value); match != nil {
			stepID = match[1]
			if source := findStep(definition, stepID); source != nil {
				use = source.Use
			}
		}
		notes = append(notes, fmt.Sprintf(parameterSourceNote, name, stepID, use))
	}
	return parameters, notes
}

// translateParameter turns a value that is exactly one step's outputs into
// the source of that step.
func translateParameter(definition *pipeline.WorkflowDefinition, value string) (SourceSpec, bool) {
	match := stepOutputReference.FindStringSubmatch(value)
	if match == nil {
		return SourceSpec{}, false
	}
	source := findStep(definition, match[1])
	if source == nil {
		return SourceSpec{}, false
	}
	with := source.With.Payload
	switch source.Use {
	case stepUseCredentialOffer:
		return SourceSpec{
			Kind:       SourceCredentialOffer,
			Identifier: stringValue(with[credentialIDKey]),
		}, true
	case stepUseVerificationDeeplink:
		return SourceSpec{
			Kind:       SourceUseCaseVerification,
			Identifier: stringValue(with[useCaseIDKey]),
		}, true
	case stepUseHTTPRequest:
		request := &HTTPSourceRequest{
			Method: stringValue(with["method"]),
			URL:    stringValue(with["url"]),
			Body:   with["body"],
		}
		if headers, ok := with["headers"].(map[string]any); ok && len(headers) > 0 {
			request.Headers = make(map[string]string, len(headers))
			for name, header := range headers {
				request.Headers[name] = fmt.Sprint(header)
			}
		}
		valuePath := ""
		if _, rest, ok := strings.Cut(value, httpBodyOutputMarker); ok {
			valuePath = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(rest), "}}"))
		}
		return SourceSpec{Kind: SourceHTTP, Request: request, ValuePath: valuePath}, true
	default:
		return SourceSpec{}, false
	}
}

func findStep(definition *pipeline.WorkflowDefinition, id string) *pipeline.StepDefinition {
	for i := range definition.Steps {
		if definition.Steps[i].ID == id {
			return &definition.Steps[i]
		}
	}
	return nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func isInvalid(err error) bool {
	return errors.Is(err, ErrInvalid)
}
