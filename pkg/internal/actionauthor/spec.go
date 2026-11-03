// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

// PrereqRule says whether a slot's action starts from prerequisite actions.
type PrereqRule int

// Prerequisite rules of the slots.
const (
	PrereqForbidden PrereqRule = iota
	PrereqRequired
	PrereqOptional
)

// SlotRule describes one action category the agent can author.
type SlotRule struct {
	Category      string
	Prerequisites PrereqRule
	Rejection     bool
}

// Slots lists the action categories the agent can author. install-app has
// no slot: the agent never installs apps.
var Slots = map[string]SlotRule{
	"onboarding": {Category: "onboarding", Prerequisites: PrereqForbidden},
	"get-credential": {
		Category:      "get-credential-generic",
		Prerequisites: PrereqRequired,
		Rejection:     true,
	},
	"get-credential-specific": {
		Category:      "get-credential-specific",
		Prerequisites: PrereqRequired,
		Rejection:     true,
	},
	"verify-credential": {
		Category:      "verify-credential",
		Prerequisites: PrereqRequired,
		Rejection:     true,
	},
	"other": {
		Category:      "other",
		Prerequisites: PrereqOptional,
		Rejection:     true,
	},
}

// Outcomes of an authored action.
const (
	OutcomeSuccess   = "success"
	OutcomeRejection = "rejection"
)

// Parameter source kinds.
const (
	SourceCredentialOffer     = "credential_offer"
	SourceUseCaseVerification = "use_case_verification"
	SourceHTTP                = "http"
	SourceLiteral             = "literal"
)

const (
	maxPrerequisites   = 5
	maxParameters      = 8
	maxLiteralLength   = 1000
	maxHTTPBodyBytes   = 16 * 1024
	httpsScheme        = "https://"
	allowedHeadersRule = "only Content-Type, Accept and Accept-Language are allowed; " +
		"credentials must not be stored in Temporal history"
	parameterNameRule  = "^[A-Za-z][A-Za-z0-9_]{0,63}$"
	actionParameterRef = `\$\{([A-Za-z][A-Za-z0-9_]{0,63})\}`
)

var (
	parameterName   = regexp.MustCompile(parameterNameRule)
	actionParameter = regexp.MustCompile(actionParameterRef)
	statusReference = regexp.MustCompile(`\$\{([^}]*)\}`)
	// allowedHeaders are the http source request headers, lower case: the
	// session input is stored in Temporal history, so no header may carry a
	// credential.
	allowedHeaders = map[string]bool{
		"content-type":    true,
		"accept":          true,
		"accept-language": true,
	}
)

// Spec describes the action a session authors: its slot, expected outcome,
// starting state and runtime parameters.
type Spec struct {
	Slot          string             `json:"slot"`
	Outcome       string             `json:"outcome,omitempty"` // "success" (default) | "rejection"
	Prerequisites []PrerequisiteSpec `json:"prerequisites,omitempty"`
	Parameters    []ParameterSpec    `json:"parameters,omitempty"`
}

// PrerequisiteSpec is a stored action that builds the starting state, with
// values for its own parameters.
type PrerequisiteSpec struct {
	Action     string          `json:"action"` // canonified wallet action path org/wallet/action
	Parameters []ParameterSpec `json:"parameters,omitempty"`
}

// ParameterSpec names a runtime parameter and the source of its fresh values.
type ParameterSpec struct {
	Name   string     `json:"name"`
	Source SourceSpec `json:"source"`
}

// SourceSpec generates a parameter value: a Credimi credential offer, a
// use-case verification, a declared HTTP call or a fixed value.
type SourceSpec struct {
	Kind       string             `json:"kind"` // credential_offer | use_case_verification | http | literal
	Identifier string             `json:"identifier,omitempty"`
	Value      string             `json:"value,omitempty"`
	Request    *HTTPSourceRequest `json:"request,omitempty"`
	ValuePath  string             `json:"value_path,omitempty"`
	Captures   map[string]string  `json:"captures,omitempty"`
	Status     *StatusSpec        `json:"status,omitempty"`
}

// HTTPSourceRequest is the HTTP call of an http source.
type HTTPSourceRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`
}

// StatusSpec is the protocol session status an http source's issuer or
// verifier must reach after the action ran.
type StatusSpec struct {
	URL       string   `json:"url"` // may contain ${capture}
	Path      string   `json:"path"`
	Success   []string `json:"success,omitempty"`
	Rejection []string `json:"rejection,omitempty"`
}

// Session is decoded from the session workflow's start input.
type Session struct {
	BuildRequest
	Spec
}

// IsEmpty reports whether no field of the spec is set: the session's plan is
// then proposed in the chat rather than fixed at start.
func (s Spec) IsEmpty() bool {
	return s.Slot == "" && s.Outcome == "" &&
		len(s.Prerequisites) == 0 && len(s.Parameters) == 0
}

// PlanSHA256 hashes the JSON encoding of a normalized spec.
func PlanSHA256(s Spec) string {
	encoded, err := json.Marshal(s)
	if err != nil {
		// A Spec holds only JSON-encodable values; http bodies come from JSON.
		panic(fmt.Sprintf("encode plan: %v", err))
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

// ParameterInfo describes a runtime parameter of the authored action; its
// values are generated by Credimi, never shown to the agent.
type ParameterInfo struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	HasStatus   bool   `json:"has_status"`
}

// ParameterDescription tells the agent what a parameter's values are,
// without revealing them.
func ParameterDescription(src SourceSpec) string {
	switch src.Kind {
	case SourceCredentialOffer:
		return "credential offer deeplink of " + src.Identifier
	case SourceUseCaseVerification:
		return "presentation request deeplink of " + src.Identifier
	case SourceHTTP:
		if src.Request == nil {
			return src.ValuePath
		}
		return src.Request.Method + " " + src.Request.URL + " → " + src.ValuePath
	default:
		return "fixed value"
	}
}

// ParameterInfos describes the spec's own parameters, in order.
func ParameterInfos(s Spec) []ParameterInfo {
	infos := make([]ParameterInfo, 0, len(s.Parameters))
	for _, parameter := range s.Parameters {
		infos = append(infos, ParameterInfo{
			Name:        parameter.Name,
			Kind:        parameter.Source.Kind,
			Description: ParameterDescription(parameter.Source),
			HasStatus:   parameter.Source.Status != nil,
		})
	}
	return infos
}

// ResolvedPrerequisite is a prerequisite action the agent runs before
// exploring and before every replay.
type ResolvedPrerequisite struct {
	Action     string   `json:"action"`
	Name       string   `json:"name"`
	Code       string   `json:"code"`
	Parameters []string `json:"parameters"`
}

// ResolvePrerequisites checks the spec's records like CheckSpecRecords and
// returns its prerequisite actions in order, with their code.
func ResolvePrerequisites(
	app core.App,
	selection Selection,
	spec *Spec,
) ([]ResolvedPrerequisite, error) {
	records, err := CheckSpecRecords(app, selection, spec)
	if err != nil {
		return nil, err
	}
	resolved := make([]ResolvedPrerequisite, 0, len(records))
	for i, record := range records {
		resolved = append(resolved, ResolvedPrerequisite{
			Action:     spec.Prerequisites[i].Action,
			Name:       record.GetString("name"),
			Code:       record.GetString("code"),
			Parameters: ActionParameters(record.GetString("code")),
		})
	}
	return resolved, nil
}

func invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalid}, args...)...)
}

// SlotCategory returns the wallet_actions category of a slot.
func SlotCategory(slot string) string {
	return Slots[slot].Category
}

func slotNames() string {
	names := make([]string, 0, len(Slots))
	for name := range Slots {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// Normalize sets the default outcome and validates the spec.
func (s *Spec) Normalize() error {
	if s.Outcome == "" {
		s.Outcome = OutcomeSuccess
	}
	rule, ok := Slots[s.Slot]
	if !ok {
		return invalidf("slot: must be one of %s", slotNames())
	}
	switch s.Outcome {
	case OutcomeSuccess:
	case OutcomeRejection:
		if !rule.Rejection {
			return invalidf("outcome: the %s slot has no rejection outcome", s.Slot)
		}
	default:
		return invalidf("outcome: must be success or rejection")
	}
	switch {
	case rule.Prerequisites == PrereqForbidden && len(s.Prerequisites) > 0:
		return invalidf("prerequisites: the %s slot takes no prerequisites", s.Slot)
	case rule.Prerequisites == PrereqRequired && len(s.Prerequisites) == 0:
		return invalidf("prerequisites: the %s slot needs at least one prerequisite", s.Slot)
	case len(s.Prerequisites) > maxPrerequisites:
		return invalidf("prerequisites: at most %d", maxPrerequisites)
	}
	for i := range s.Prerequisites {
		field := fmt.Sprintf("prerequisites[%d]", i)
		prerequisite := &s.Prerequisites[i]
		prerequisite.Action = canonify.NormalizePath(prerequisite.Action)
		if prerequisite.Action == "" {
			return invalidf("%s.action: required", field)
		}
		if err := validateParameters(field+".parameters", prerequisite.Parameters, ""); err != nil {
			return err
		}
	}
	return validateParameters("parameters", s.Parameters, s.Outcome)
}

// validateParameters checks one parameter list; outcome is empty for
// prerequisite lists, which cannot check a protocol status.
func validateParameters(field string, parameters []ParameterSpec, outcome string) error {
	if len(parameters) > maxParameters {
		return invalidf("%s: at most %d", field, maxParameters)
	}
	seen := make(map[string]bool, len(parameters))
	for i, parameter := range parameters {
		item := fmt.Sprintf("%s[%d]", field, i)
		if !parameterName.MatchString(parameter.Name) {
			return invalidf("%s.name: must match %s", item, parameterNameRule)
		}
		if seen[parameter.Name] {
			return invalidf("%s.name: duplicate name %s", item, parameter.Name)
		}
		seen[parameter.Name] = true
		if err := validateSource(item+".source", parameter.Source, outcome); err != nil {
			return err
		}
	}
	return nil
}

func validateSource(field string, source SourceSpec, outcome string) error {
	switch source.Kind {
	case SourceCredentialOffer, SourceUseCaseVerification:
		if source.Identifier == "" {
			return invalidf("%s.identifier: required", field)
		}
		if source.Value != "" || source.Request != nil || source.ValuePath != "" ||
			len(source.Captures) > 0 || source.Status != nil {
			return invalidf("%s: kind %s takes only an identifier", field, source.Kind)
		}
		return nil
	case SourceLiteral:
		if n := utf8.RuneCountInString(source.Value); n < 1 || n > maxLiteralLength {
			return invalidf("%s.value: must be 1-%d characters", field, maxLiteralLength)
		}
		return nil
	case SourceHTTP:
		return validateHTTPSource(field, source, outcome)
	default:
		return invalidf(
			"%s.kind: must be credential_offer, use_case_verification, http or literal",
			field,
		)
	}
}

func validateHTTPSource(field string, source SourceSpec, outcome string) error {
	request := source.Request
	if request == nil {
		return invalidf("%s.request: required", field)
	}
	if request.Method != "GET" && request.Method != "POST" {
		return invalidf("%s.request.method: must be GET or POST", field)
	}
	if !isHTTPSURL(request.URL) {
		return invalidf("%s.request.url: must be an https URL", field)
	}
	for name := range request.Headers {
		if !allowedHeaders[strings.ToLower(name)] {
			return invalidf("%s.request.headers: %s", field, allowedHeadersRule)
		}
	}
	if request.Body != nil {
		body, err := json.Marshal(request.Body)
		if err != nil {
			return invalidf("%s.request.body: must be JSON", field)
		}
		if len(body) > maxHTTPBodyBytes {
			return invalidf("%s.request.body: must be at most 16 KiB", field)
		}
	}
	if source.ValuePath == "" {
		return invalidf("%s.value_path: required", field)
	}
	for name, path := range source.Captures {
		if !parameterName.MatchString(name) {
			return invalidf("%s.captures.%s: name must match %s", field, name, parameterNameRule)
		}
		if path == "" {
			return invalidf("%s.captures.%s: path required", field, name)
		}
	}
	if source.Status == nil {
		return nil
	}
	return validateStatus(field+".status", source, outcome)
}

func validateStatus(field string, source SourceSpec, outcome string) error {
	status := source.Status
	if outcome == "" {
		return invalidf("%s: only the new action's parameters can check a protocol status", field)
	}
	host, err := StatusTemplateHost(status.URL)
	if err != nil {
		return invalidf("%s.url: %v", field, err)
	}
	if strings.Contains(host, statusPlaceholder) {
		return invalidf("%s.url: the host must not use captures", field)
	}
	for _, match := range statusReference.FindAllStringSubmatch(status.URL, -1) {
		if _, ok := source.Captures[match[1]]; !ok {
			return invalidf("%s.url: ${%s} is not a capture", field, match[1])
		}
	}
	if status.Path == "" {
		return invalidf("%s.path: required", field)
	}
	if outcome == OutcomeSuccess && len(status.Success) == 0 {
		return invalidf("%s.success: required for a success outcome", field)
	}
	if outcome == OutcomeRejection && len(status.Rejection) == 0 {
		return invalidf("%s.rejection: required for a rejection outcome", field)
	}
	return nil
}

// statusPlaceholder stands for a ${capture} while a status URL template is parsed.
const statusPlaceholder = "credimi-capture"

// StatusTemplateHost returns the host of a status URL template, its
// ${capture} references replaced by a placeholder. Validation refuses a host
// that contains the placeholder, so the check activity can require the
// filled URL to keep exactly this host.
func StatusTemplateHost(template string) (string, error) {
	parsed, err := url.Parse(statusReference.ReplaceAllString(template, statusPlaceholder))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", errors.New("must be an https URL")
	}
	return parsed.Host, nil
}

func isHTTPSURL(raw string) bool {
	if !strings.HasPrefix(raw, httpsScheme) {
		return false
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Host != ""
}

// LookupPath reads a dot-separated path in decoded JSON; numeric segments
// index arrays. It returns strings and numbers only.
func LookupPath(v any, path string) (string, bool) {
	if path == "" {
		return "", false
	}
	current := v
	for _, segment := range strings.Split(path, ".") {
		switch node := current.(type) {
		case map[string]any:
			next, ok := node[segment]
			if !ok {
				return "", false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(node) {
				return "", false
			}
			current = node[index]
		default:
			return "", false
		}
	}
	switch value := current.(type) {
	case string:
		return value, true
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64), true
	case json.Number:
		return value.String(), true
	default:
		return "", false
	}
}

// ActionParameters returns the sorted unique bare ${name} references of an
// action YAML, minus the keys of its header env map.
func ActionParameters(code string) []string {
	var header struct {
		Env map[string]any `yaml:"env"`
	}
	_ = yaml.NewDecoder(strings.NewReader(code)).Decode(&header)
	seen := map[string]bool{}
	names := []string{}
	for _, match := range actionParameter.FindAllStringSubmatch(code, -1) {
		name := match[1]
		if _, local := header.Env[name]; local || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// CheckRecordAccess resolves a canonified path to a record of the collection
// that is published or owned by the organization.
func CheckRecordAccess(app core.App, collection, path, orgID string) (*core.Record, error) {
	record, err := resolveCollectionPath(app, collection, canonify.NormalizePath(path))
	if errors.Is(err, sql.ErrNoRows) ||
		(err == nil && !record.GetBool("published") && record.GetString("owner") != orgID) {
		return nil, invalidf("%s not found", path)
	}
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", path, err)
	}
	return record, nil
}

// resolveCollectionPath walks the canonified path of one collection only:
// canonify.Resolve picks any collection with that path length.
func resolveCollectionPath(app core.App, collection, path string) (*core.Record, error) {
	var chain []string
	for current := collection; ; {
		tpl, ok := canonify.CanonifyPaths[current]
		if !ok {
			return nil, sql.ErrNoRows
		}
		chain = append([]string{current}, chain...)
		if tpl.Parent == nil {
			break
		}
		current = tpl.Parent.Collection
	}
	segments := strings.Split(path, "/")
	if len(segments) != len(chain) {
		return nil, sql.ErrNoRows
	}
	var record *core.Record
	for i, name := range chain {
		tpl := canonify.CanonifyPaths[name]
		filter := dbx.HashExp{tpl.CanonifiedField: segments[i]}
		if i > 0 {
			filter[tpl.Parent.Field] = record.Id
		}
		records, err := app.FindAllRecords(name, filter)
		if err != nil {
			return nil, err
		}
		if len(records) != 1 || segments[i] == "" {
			return nil, sql.ErrNoRows
		}
		record = records[0]
	}
	return record, nil
}

// CheckSpecRecords checks that every record the spec names is accessible to
// the selected organization, and returns the prerequisite actions in order.
// Prerequisites must be actions of the selected wallet and receive exactly
// the parameters their code references.
func CheckSpecRecords(app core.App, selection Selection, spec *Spec) ([]*core.Record, error) {
	orgID := selection.Organization.Id
	prerequisites := make([]*core.Record, 0, len(spec.Prerequisites))
	for _, prerequisite := range spec.Prerequisites {
		record, err := CheckRecordAccess(app, "wallet_actions", prerequisite.Action, orgID)
		if err != nil {
			return nil, err
		}
		if record.GetString("wallet") != selection.Wallet.Id {
			return nil, invalidf("prerequisite %s belongs to another wallet", prerequisite.Action)
		}
		needed := ActionParameters(record.GetString("code"))
		given := ParameterNames(prerequisite.Parameters)
		sort.Strings(given)
		if strings.Join(needed, "\x00") != strings.Join(given, "\x00") {
			return nil, invalidf(
				"prerequisite %s needs parameters %v, got %v",
				prerequisite.Action,
				needed,
				given,
			)
		}
		if err := checkSourceRecords(app, prerequisite.Parameters, orgID); err != nil {
			return nil, err
		}
		prerequisites = append(prerequisites, record)
	}
	if err := checkSourceRecords(app, spec.Parameters, orgID); err != nil {
		return nil, err
	}
	return prerequisites, nil
}

// ParameterNames returns the names of a parameter list, in order.
func ParameterNames(parameters []ParameterSpec) []string {
	names := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		names = append(names, parameter.Name)
	}
	return names
}

// sourceCollections maps the record-backed source kinds to their collection.
var sourceCollections = map[string]string{
	SourceCredentialOffer:     "credentials",
	SourceUseCaseVerification: "use_cases_verifications",
}

func checkSourceRecords(app core.App, parameters []ParameterSpec, orgID string) error {
	for _, parameter := range parameters {
		collection, ok := sourceCollections[parameter.Source.Kind]
		if !ok {
			continue
		}
		if _, err := CheckRecordAccess(
			app,
			collection,
			parameter.Source.Identifier,
			orgID,
		); err != nil {
			return err
		}
	}
	return nil
}
