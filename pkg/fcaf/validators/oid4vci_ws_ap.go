// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"mime"
	"net/url"
	"slices"
	"strings"
	"time"
)

// HTTPHeaderCountValidator requires a header field to occur a given number of
// times in the Wallet's requests, counted by Capture from the raw header list
// (header_counts): joined header values cannot show that a field was sent once.
type HTTPHeaderCountValidator struct{}

func (HTTPHeaderCountValidator) ID() string { return "http.header_count" }

func (HTTPHeaderCountValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Request string `json:"request"`
		Header  string `json:"header"`
		Count   *int   `json:"count"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Request == "" || params.Header == "" || params.Count == nil {
		return Result{Status: StatusError, Message: "request, header and count params are required"}
	}
	exchange, err := decodeOID4VCIExchange(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	requests := exchange.sessionRequestsEndingIn(params.Request)
	if len(requests) == 0 {
		return exchange.missingEvidence(fmt.Sprintf(
			"the issuance session records no Wallet request ending in %s",
			params.Request,
		))
	}
	name := strings.ToLower(params.Header)
	for _, request := range requests {
		counts, ok := request.raw["header_counts"].(map[string]any)
		if !ok {
			return Result{
				Status: StatusInconclusive,
				Message: fmt.Sprintf(
					"the %s request records no header_counts, so the occurrences are not evidenced",
					request.path,
				),
			}
		}
		count := 0
		if value, ok := counts[name].(float64); ok {
			count = int(value)
		}
		if count != *params.Count {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"the %s %s request carries %d %s header field(s), expected %d",
					request.method,
					request.path,
					count,
					params.Header,
					*params.Count,
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"%d request(s) ending in %s carry exactly %d %s header field(s)",
			len(requests),
			params.Request,
			*params.Count,
			params.Header,
		),
	}
}

// JSONValueShapeValidator checks the JSON type of selected values, the length
// of an array and the type of its items. Without any constraint it requires
// the selection to exist, which is how a parameter's presence is evidenced.
type JSONValueShapeValidator struct{}

func (JSONValueShapeValidator) ID() string { return "json.value_shape" }

func (JSONValueShapeValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Type      string `json:"type"`
		Length    *int   `json:"length"`
		ItemsType string `json:"items_type"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	for _, name := range []string{params.Type, params.ItemsType} {
		if name != "" && !slices.Contains(jsonTypeNames, name) {
			return Result{
				Status:  StatusError,
				Message: fmt.Sprintf("JSON type %q is unknown", name),
			}
		}
	}
	selected, result := resolveSelection(input)
	if result != nil {
		return *result
	}
	for _, item := range selected {
		if params.Type != "" && jsonTypeName(item.value) != params.Type {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"%s is a JSON %s, expected %s",
					item.label,
					jsonTypeName(item.value),
					params.Type,
				),
			}
		}
		if params.Length == nil && params.ItemsType == "" {
			continue
		}
		elements, ok := item.value.([]any)
		if !ok {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"%s is a JSON %s, expected array",
					item.label,
					jsonTypeName(item.value),
				),
			}
		}
		if params.Length != nil && len(elements) != *params.Length {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"%s has %d element(s), expected %d",
					item.label,
					len(elements),
					*params.Length,
				),
			}
		}
		for index, element := range elements {
			if params.ItemsType != "" && jsonTypeName(element) != params.ItemsType {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"%s[%d] is a JSON %s, expected %s",
						item.label,
						index,
						jsonTypeName(element),
						params.ItemsType,
					),
				}
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("%d selected value(s) have the required shape", len(selected)),
	}
}

// URISyntaxValidator requires selected strings to be syntactically correct
// RFC 3986 URIs: a scheme, only permitted characters, valid percent-encoding,
// and at most one fragment delimiter.
type URISyntaxValidator struct{}

func (URISyntaxValidator) ID() string { return "uri.syntax" }

func (URISyntaxValidator) Validate(_ context.Context, input Input) Result {
	selected, result := resolveSelection(input)
	if result != nil {
		return *result
	}
	for _, item := range selected {
		text, result := selectedString(item)
		if result != nil {
			return *result
		}
		if problem := rfc3986URIProblem(text); problem != "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("%s %q %s", item.label, text, problem),
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("%d value(s) are syntactically correct RFC 3986 URIs", len(selected)),
	}
}

// rfc3986URIProblem names the first RFC 3986 rule a URI breaks, or returns "".
func rfc3986URIProblem(raw string) string {
	if raw == "" {
		return "is empty"
	}
	end, ok := statusURISchemeEnd(raw)
	if !ok {
		return "has no RFC 3986 scheme"
	}
	if end == len(raw) {
		return "has an empty hierarchical part"
	}
	if statusURIHasIllegalCharacter(raw) {
		return "contains a character outside the RFC 3986 set"
	}
	if !statusURIPercentEncodingValid(raw) {
		return "contains an invalid percent-encoding"
	}
	if strings.Count(raw, "#") > 1 {
		return "contains more than one fragment delimiter"
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "does not parse: " + err.Error()
	}
	if parsed.Host == "" && strings.ContainsAny(parsed.Opaque+parsed.Path, "[]") {
		return "uses [ or ] outside an IP literal host"
	}
	return ""
}

// requestValueRef names a value of the Wallet's requests: the path inside
// every request ending in `request`.
type requestValueRef struct {
	Request string `json:"request"`
	Path    string `json:"path"`
}

// OID4VCIRequestValuesEqualValidator requires a value of each later request to
// equal a value of the latest earlier request, such as the request_uri of an
// Authorization Request and the one the PAR endpoint returned before it.
type OID4VCIRequestValuesEqualValidator struct{}

func (OID4VCIRequestValuesEqualValidator) ID() string { return "oid4vci.request_values_equal" }

func (OID4VCIRequestValuesEqualValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Source requestValueRef `json:"source"`
		Target requestValueRef `json:"target"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Source.Request == "" || params.Source.Path == "" ||
		params.Target.Request == "" || params.Target.Path == "" {
		return Result{Status: StatusError, Message: "source and target need request and path"}
	}
	exchange, err := decodeOID4VCIExchange(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	sources := exchange.sessionRequestsEndingIn(params.Source.Request)
	targets := exchange.sessionRequestsEndingIn(params.Target.Request)
	if len(sources) == 0 || len(targets) == 0 {
		return exchange.missingEvidence(fmt.Sprintf(
			"the issuance session records %d request(s) ending in %s and %d ending in %s",
			len(sources),
			params.Source.Request,
			len(targets),
			params.Target.Request,
		))
	}
	for _, target := range targets {
		var source *capturedRequest
		for index := range sources {
			if sources[index].at.Before(target.at) {
				source = &sources[index]
			}
		}
		if source == nil {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"the %s request at %s follows no request ending in %s",
					target.path,
					target.at.Format(time.RFC3339Nano),
					params.Source.Request,
				),
			}
		}
		expected, result := singleRequestValue(*source, params.Source.Path)
		if result != nil {
			return *result
		}
		actual, result := singleRequestValue(target, params.Target.Path)
		if result != nil {
			return *result
		}
		if expected != actual {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"%s of the %s request is %v, but %s of the preceding %s request is %v",
					params.Target.Path,
					target.path,
					actual,
					params.Source.Path,
					source.path,
					expected,
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"%d request(s) ending in %s repeat %s of the preceding request ending in %s",
			len(targets),
			params.Target.Request,
			params.Source.Path,
			params.Source.Request,
		),
	}
}

// singleRequestValue reads one scalar value of a request; a missing or
// redacted value is a Result.
func singleRequestValue(request capturedRequest, path string) (any, *Result) {
	found := walkCapturePath(request.raw, path, "the "+request.path+" request")
	if len(found) != 1 {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("the %s request records no %s", request.path, path),
		}
	}
	if isRedactionMarker(found[0].value) {
		return nil, &Result{
			Status: StatusInconclusive,
			Message: fmt.Sprintf(
				"%s of the %s request is redacted; only a session created with evidence_mode \"full\" records it",
				path,
				request.path,
			),
		}
	}
	return found[0].value, nil
}

// OID4VCIIssuerResponseValidator confirms that the issuer delivered the
// response a test scenario sets up, for instance a PAR response without
// request_uri. It checks the test harness, not the Wallet: when the response
// does not match, the Wallet's reaction is no evidence of the test, so the
// result is blocked.
type OID4VCIIssuerResponseValidator struct{}

func (OID4VCIIssuerResponseValidator) ID() string { return "oid4vci.issuer_response" }

func (OID4VCIIssuerResponseValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Request   string            `json:"request"`
		Status    int               `json:"status"`
		MediaType string            `json:"media_type"`
		Required  []string          `json:"required_members"`
		Absent    []string          `json:"absent_members"`
		Equals    map[string]string `json:"members_equal"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Request == "" {
		return Result{Status: StatusError, Message: "request param is required"}
	}
	exchange, err := decodeOID4VCIExchange(input.Value)
	if err != nil {
		return Result{Status: StatusBlocked, Message: err.Error()}
	}
	requests := exchange.sessionRequestsEndingIn(params.Request)
	if len(requests) == 0 {
		return Result{
			Status: StatusBlocked,
			Message: fmt.Sprintf(
				"the Wallet sent no request ending in %s, so the issuer response the test sets up was never delivered",
				params.Request,
			),
		}
	}
	blocked := func(request capturedRequest, format string, args ...any) Result {
		return Result{
			Status: StatusBlocked,
			Message: fmt.Sprintf("the %s response ", request.path) + fmt.Sprintf(format, args...) +
				"; it is not the response the test scenario requires",
		}
	}
	for _, request := range requests {
		response := request.response
		if response == nil {
			return blocked(request, "was not recorded")
		}
		status, _ := response["status"].(float64)
		if params.Status != 0 && int(status) != params.Status {
			return blocked(request, "has status %v, expected %d", response["status"], params.Status)
		}
		if params.MediaType != "" {
			contentType, _ := response["content_type"].(string)
			mediaType, _, err := mime.ParseMediaType(contentType)
			if err != nil || !strings.EqualFold(mediaType, params.MediaType) {
				return blocked(
					request,
					"has Content-Type %q, expected %s",
					contentType,
					params.MediaType,
				)
			}
		}
		body, _ := response["body"].(map[string]any)
		if len(params.Required)+len(params.Absent)+len(params.Equals) > 0 && body == nil {
			return blocked(request, "has no JSON body")
		}
		for _, member := range params.Required {
			if !memberPresent(body, member) {
				return blocked(request, "has no %s", member)
			}
		}
		for _, member := range params.Absent {
			if _, present := body[member]; present {
				return blocked(request, "carries %s", member)
			}
		}
		for _, member := range sortedKeys(params.Equals) {
			if value, _ := body[member].(string); value != params.Equals[member] {
				return blocked(
					request,
					"has %s %v, expected %q",
					member,
					body[member],
					params.Equals[member],
				)
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"the issuer answered %d request(s) ending in %s with the response the test sets up",
			len(requests),
			params.Request,
		),
	}
}

// OID4VCINonceFromNonceEndpointValidator requires the nonce of the Wallet's
// proofs of one type to be a c_nonce the Nonce Endpoint returned before the
// Credential Request. Capture attributes a nonce response to the session when a
// Credential Request proof repeats its c_nonce.
type OID4VCINonceFromNonceEndpointValidator struct{}

func (OID4VCINonceFromNonceEndpointValidator) ID() string {
	return "oid4vci.nonce_from_nonce_endpoint"
}

func (OID4VCINonceFromNonceEndpointValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		ProofType string `json:"proof_type"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.ProofType != "jwt" && params.ProofType != "attestation" {
		return Result{Status: StatusError, Message: "proof_type must be jwt or attestation"}
	}
	exchange, err := decodeOID4VCIExchange(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	selected, result := captureSelector{
		Path:    "raw.credential_request_proofs." + params.ProofType + "[].payload.nonce",
		Require: "raw.credential_request_proofs",
		Absent:  StatusNotApplicable,
		Reason: fmt.Sprintf(
			"the Wallet sent no %s proof, and the source applies to a Wallet that uses the %s proof type",
			params.ProofType,
			params.ProofType,
		),
	}.resolve(input.Value)
	if result != nil {
		return *result
	}
	raw, _ := exchange.session["raw"].(map[string]any)
	proofs, _ := raw["credential_request_proofs"].(map[string]any)
	requestAt, err := parseCaptureTime(proofs["at"])
	if err != nil {
		return Result{Status: StatusFail, Message: "the Credential Request proofs record no time"}
	}
	var nonces []string
	for _, request := range exchange.sessionRequestsEndingIn("/nonce") {
		body, _ := request.response["body"].(map[string]any)
		if value, ok := body["c_nonce"].(string); ok && !request.at.After(requestAt) {
			nonces = append(nonces, value)
		}
	}
	if len(nonces) == 0 {
		return Result{
			Status: StatusFail,
			Message: "no Nonce Endpoint response preceding the Credential Request carries a c_nonce " +
				"a proof repeats, so the Wallet did not use a Nonce Endpoint c_nonce",
		}
	}
	for _, item := range selected {
		nonce, result := selectedString(item)
		if result != nil {
			return *result
		}
		if !slices.Contains(nonces, nonce) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"%s is not the c_nonce of any of the %d preceding Nonce Endpoint response(s)",
					item.label,
					len(nonces),
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"the nonce of %d %s proof(s) is a c_nonce the Nonce Endpoint returned before the Credential Request",
			len(selected),
			params.ProofType,
		),
	}
}

// requestFilter selects unattributed ledger requests that may belong to the
// Wallet: the issuer's host and the given path prefixes or suffixes. Ledger
// requests carry no session, so a filter is an assumption that the deployment
// served no other client for these paths in the window.
type requestFilter struct {
	PathPrefixes []string `json:"path_prefixes"`
	PathSuffixes []string `json:"path_suffixes"`
}

func (f requestFilter) empty() bool {
	return len(f.PathPrefixes)+len(f.PathSuffixes) == 0
}

func (f requestFilter) matches(request capturedRequest, issuerPath string, host string) bool {
	if strings.HasPrefix(request.path, chainedAuthorizationServerPrefix) {
		return false
	}
	if requestHost, ok := caseInsensitiveHeader(request.headers, "host"); ok && host != "" &&
		!strings.EqualFold(strings.Split(requestHost, ":")[0], host) {
		return false
	}
	for _, prefix := range f.PathPrefixes {
		if strings.HasPrefix(request.path, prefix) {
			return true
		}
	}
	for _, suffix := range f.PathSuffixes {
		if strings.HasPrefix(request.path, issuerPath) && strings.HasSuffix(request.path, suffix) {
			return true
		}
	}
	return false
}

// laterRequests returns the Wallet requests after `after`: the session's own
// (ending in one of suffixes, or any when none is given) and the unattributed
// ones the filter matches.
func (e oid4vciExchange) laterRequests(
	after time.Time,
	suffixes []string,
	unattributed requestFilter,
) ([]capturedRequest, []capturedRequest) {
	var session []capturedRequest
	for _, request := range e.requests {
		if !request.at.After(after) ||
			strings.HasPrefix(request.path, chainedAuthorizationServerPrefix) {
			continue
		}
		if len(suffixes) == 0 || slices.ContainsFunc(suffixes, func(suffix string) bool {
			return strings.HasSuffix(request.path, suffix)
		}) {
			session = append(session, request)
		}
	}
	var ledger []capturedRequest
	if !unattributed.empty() {
		configuration, _ := e.session["issuer_configuration_id"].(string)
		host := ""
		if issuer, _ := e.session["issuer_identifier"].(string); issuer != "" {
			if parsed, err := url.Parse(issuer); err == nil {
				host = parsed.Hostname()
			}
		}
		for _, request := range e.unattributed {
			if request.at.After(after) &&
				unattributed.matches(request, "/issuers/"+configuration+"/", host) {
				ledger = append(ledger, request)
			}
		}
	}
	return session, ledger
}

func describeRequests(requests []capturedRequest) string {
	described := make([]string, 0, len(requests))
	for _, request := range requests {
		described = append(described, fmt.Sprintf(
			"%s %s at %s",
			request.method,
			request.path,
			request.at.Format(time.RFC3339Nano),
		))
	}
	return strings.Join(described, ", ")
}

// OID4VCINoRequestAfterValidator requires the Wallet to make no further
// request after an anchor request, such as the Credential Request an issuer
// denied. Requests the session cannot hold, like an Authorization Request
// without request_uri, are looked up among the unattributed requests of the
// session window.
type OID4VCINoRequestAfterValidator struct{}

func (OID4VCINoRequestAfterValidator) ID() string { return "oid4vci.no_request_after" }

func (OID4VCINoRequestAfterValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		After          string        `json:"after"`
		PathSuffixes   []string      `json:"path_suffixes"`
		Unattributed   requestFilter `json:"unattributed"`
		ObserveSeconds *float64      `json:"observe_seconds"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.After == "" {
		return Result{Status: StatusError, Message: "after param is required"}
	}
	exchange, err := decodeOID4VCIExchange(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	anchors := exchange.sessionRequestsEndingIn(params.After)
	if len(anchors) == 0 {
		return Result{
			Status: StatusBlocked,
			Message: fmt.Sprintf(
				"the Wallet sent no request ending in %s, so the step the test observes never happened",
				params.After,
			),
		}
	}
	anchor := anchors[len(anchors)-1]
	session, ledger := exchange.laterRequests(anchor.at, params.PathSuffixes, params.Unattributed)
	if len(session)+len(ledger) > 0 {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"after the %s request at %s the Wallet sent %s",
				anchor.path,
				anchor.at.Format(time.RFC3339Nano),
				describeRequests(append(session, ledger...)),
			),
		}
	}
	if result := requireObservation(exchange, anchor.at, params.ObserveSeconds); result != nil {
		return *result
	}
	message := fmt.Sprintf(
		"the Wallet sent no further request after the %s request at %s",
		anchor.path,
		anchor.at.Format(time.RFC3339Nano),
	)
	if !params.Unattributed.empty() {
		message += "; unattributed requests of the window were read as the Wallet's, " +
			"assuming no other client used the deployment"
	}
	return Result{Status: StatusPass, Message: message}
}

// requireObservation makes the absence of a later request depend on evidence
// gathered long enough after the anchor: the session requests window must end
// observe_seconds (default 5) after it. A recorded later request needs no
// window, so callers check for one first; only the pass verdict is gated. The
// window ends when Capture answered GET /sessions/{id}/requests.
func requireObservation(exchange oid4vciExchange, after time.Time, observe *float64) *Result {
	seconds := 5.0
	if observe != nil {
		seconds = *observe
	}
	if exchange.window == nil {
		return &Result{
			Status:  StatusInconclusive,
			Message: "the exchange has no session_requests window, so the absence of later requests is not evidenced",
		}
	}
	if exchange.window.to.Sub(after) < time.Duration(seconds*float64(time.Second)) {
		return &Result{
			Status: StatusInconclusive,
			Message: fmt.Sprintf(
				"the requests were read %s after %s, less than the %.0f s the absence of a request needs",
				exchange.window.to.Sub(after),
				after.Format(time.RFC3339Nano),
				seconds,
			),
		}
	}
	return nil
}
