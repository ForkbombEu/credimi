// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"
)

// The OpenID4VCI validators read a Capture issuance exchange: an object with
//
//   - capture_session: GET /sessions/{id}, whose raw.oid4vci_requests lists the
//     Wallet's requests Capture tied to the session;
//   - request_ledger: GET /oid4vci/requests, the global request ledger, which
//     also holds the requests Capture did not tie to a session (the Wallet's
//     browser Authorization Request among them);
//   - issuer_metadata and authorization_server_metadata: the metadata the
//     Wallet resolved.
//
// Capture redacts sensitive values to {redacted, present, length}; only their
// presence is evidence.

// chainedAuthorizationServerPrefix is the path of Capture's internal
// auto-approving OAuth server. The issuer's authorization endpoint redirects the
// browser there, so its requests are Capture talking to itself, not the Wallet.
const chainedAuthorizationServerPrefix = "/authorization-servers/"

// capturedRequest is one HTTP request of the Capture ledger.
type capturedRequest struct {
	at      time.Time
	method  string
	path    string
	query   map[string]any
	body    any
	headers map[string]any
}

// oid4vciExchange is a decoded Capture issuance exchange.
type oid4vciExchange struct {
	session   map[string]any
	ledger    []any
	hasLedger bool
	metadata  map[string]map[string]any
}

func decodeOID4VCIExchange(value any) (oid4vciExchange, error) {
	object, ok := normalizeJSONObject(value)
	if !ok {
		return oid4vciExchange{}, fmt.Errorf(
			"issuance exchange is %s, expected object",
			jsonTypeName(value),
		)
	}
	session, ok := normalizeJSONObject(object["capture_session"])
	if !ok {
		return oid4vciExchange{}, fmt.Errorf("issuance exchange has no capture_session object")
	}
	exchange := oid4vciExchange{session: session, metadata: map[string]map[string]any{}}
	exchange.ledger, exchange.hasLedger = object["request_ledger"].([]any)
	for _, key := range []string{"issuer_metadata", "authorization_server_metadata"} {
		if metadata, ok := normalizeJSONObject(object[key]); ok {
			exchange.metadata[key] = metadata
		}
	}
	return exchange, nil
}

func decodeCapturedRequest(value any) (capturedRequest, bool) {
	object, ok := normalizeJSONObject(value)
	if !ok {
		return capturedRequest{}, false
	}
	request := capturedRequest{body: object["body"]}
	request.method, _ = object["method"].(string)
	request.path, _ = object["path"].(string)
	request.query, _ = object["query"].(map[string]any)
	request.headers, _ = object["headers"].(map[string]any)
	if at, ok := object["at"].(string); ok {
		request.at, _ = time.Parse(time.RFC3339Nano, at)
	}
	return request, request.path != ""
}

// sessionRequests returns the requests Capture tied to the session, in order.
func (e oid4vciExchange) sessionRequests() []capturedRequest {
	raw, _ := e.session["raw"].(map[string]any)
	entries, _ := raw["oid4vci_requests"].([]any)
	requests := make([]capturedRequest, 0, len(entries))
	for _, entry := range entries {
		if request, ok := decodeCapturedRequest(entry); ok {
			requests = append(requests, request)
		}
	}
	return requests
}

// walletRequest reports whether a request path is a Wallet request ending in
// suffix, excluding Capture's chained OAuth server.
func walletRequest(path, suffix string) bool {
	return strings.HasSuffix(path, suffix) &&
		!strings.HasPrefix(path, chainedAuthorizationServerPrefix)
}

func (e oid4vciExchange) sessionRequestsEndingIn(suffix string) []capturedRequest {
	var matches []capturedRequest
	for _, request := range e.sessionRequests() {
		if walletRequest(request.path, suffix) {
			matches = append(matches, request)
		}
	}
	return matches
}

// pushedAuthorizationRequest is the session's first PAR, if Capture recorded one.
func (e oid4vciExchange) pushedAuthorizationRequest() (capturedRequest, bool) {
	requests := e.sessionRequestsEndingIn("/par")
	if len(requests) == 0 {
		return capturedRequest{}, false
	}
	return requests[0], true
}

// ledgerRequestsAfterPAR attributes ledger requests ending in suffix to the
// session when Capture did not tie them to it. Candidates belong to the
// session's issuer configuration, carry no other session, and arrive between
// the session's PAR and the next request Capture tied to the session (for the
// authorization-code flow, the callback that only follows a completed
// Authorization Request). Several candidates are narrowed to the one repeating
// the PAR's state; anything else is ambiguous.
func (e oid4vciExchange) ledgerRequestsAfterPAR(suffix string) ([]capturedRequest, string, error) {
	if !e.hasLedger {
		return nil, "", fmt.Errorf("the exchange has no request_ledger")
	}
	par, ok := e.pushedAuthorizationRequest()
	if !ok || par.at.IsZero() {
		return nil, "", fmt.Errorf(
			"the session records no timed Pushed Authorization Request to anchor the ledger search",
		)
	}
	var end time.Time
	for _, request := range e.sessionRequests() {
		if request.at.After(par.at) {
			end = request.at
			break
		}
	}
	sessionID, _ := e.session["session_id"].(string)
	issuerConfiguration, _ := e.session["issuer_configuration_id"].(string)
	var candidates []capturedRequest
	var oldest time.Time
	for _, entry := range e.ledger {
		if request, ok := decodeCapturedRequest(entry); ok && !request.at.IsZero() &&
			(oldest.IsZero() || request.at.Before(oldest)) {
			oldest = request.at
		}
	}
	// The ledger is bounded: once later traffic has evicted the PAR's time, a
	// missing request is not evidence of a Wallet that sent none.
	if oldest.IsZero() || oldest.After(par.at) {
		return nil, "", fmt.Errorf(
			"the request ledger no longer reaches back to the PAR at %s",
			par.at.Format(time.RFC3339Nano),
		)
	}
	for _, entry := range e.ledger {
		object, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if owner, _ := object["session_id"].(string); owner != "" && owner != sessionID {
			continue
		}
		if configuration, _ := object["issuer_configuration_id"].(string); configuration != issuerConfiguration {
			continue
		}
		request, ok := decodeCapturedRequest(entry)
		if !ok || !walletRequest(request.path, suffix) || request.at.Before(par.at) {
			continue
		}
		if !end.IsZero() && request.at.After(end) {
			continue
		}
		candidates = append(candidates, request)
	}
	window := fmt.Sprintf("between the PAR at %s and ", par.at.Format(time.RFC3339Nano))
	if end.IsZero() {
		window += "the end of the ledger"
	} else {
		window += "the next session request at " + end.Format(time.RFC3339Nano)
	}
	if len(candidates) <= 1 {
		return candidates, "attributed from the global ledger " + window, nil
	}
	parBody, _ := par.body.(map[string]any)
	state, _ := parBody["state"].(string)
	var sameState []capturedRequest
	for _, candidate := range candidates {
		if value, _ := requestParameters(candidate, "query_or_body")["state"].(string); state != "" &&
			value == state {
			sameState = append(sameState, candidate)
		}
	}
	if len(sameState) == 1 {
		return sameState, "attributed from the global ledger by the PAR state " + window, nil
	}
	return nil, "", fmt.Errorf(
		"%d ledger requests ending in %s arrived %s and none is singled out by the PAR state",
		len(candidates),
		suffix,
		window,
	)
}

// requestParameters returns the request members of the given location:
// query, body, or query_or_body (body members win on a clash).
func requestParameters(request capturedRequest, location string) map[string]any {
	parameters := map[string]any{}
	if location == "query" || location == "query_or_body" {
		for key, value := range request.query {
			parameters[key] = value
		}
	}
	if location == "body" || location == "query_or_body" {
		if body, ok := request.body.(map[string]any); ok && !isRedactionMarker(body) {
			for key, value := range body {
				parameters[key] = value
			}
		}
	}
	return parameters
}

// OID4VCIWalletRequestValidator checks the Wallet's HTTP requests of one kind,
// located by path suffix, against the endpoint the Wallet resolved from
// metadata and against the request parameters the source requires.
type OID4VCIWalletRequestValidator struct{}

func (OID4VCIWalletRequestValidator) ID() string { return "oid4vci.wallet_request" }

type walletRequestEndpoint struct {
	Metadata string `json:"metadata"`
	Field    string `json:"field"`
}

type walletRequestParameters struct {
	In       string   `json:"in"`
	Required []string `json:"required"`
	Absent   []string `json:"absent"`
}

type walletRequestParams struct {
	PathSuffix     string                   `json:"path_suffix"`
	LedgerFallback bool                     `json:"ledger_fallback"`
	Methods        []string                 `json:"methods"`
	Endpoint       *walletRequestEndpoint   `json:"endpoint"`
	RequireTLS     bool                     `json:"require_tls"`
	ForbidQuery    bool                     `json:"forbid_query"`
	MediaType      string                   `json:"media_type"`
	Parameters     *walletRequestParameters `json:"parameters"`
}

// Validate locates every Wallet request whose path ends in params.path_suffix
// in the session (or, with ledger_fallback, attributed from the global ledger)
// and requires each to pass every configured check:
//
//   - methods: the HTTP method is one of these;
//   - endpoint: scheme host, port, path and query members of the metadata
//     endpoint URL are those the request used;
//   - require_tls: the request reached Capture over https;
//   - forbid_query: the request carries no query component;
//   - media_type: the Content-Type media type;
//   - parameters: members required or absent in the query, body or both.
func (OID4VCIWalletRequestValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[walletRequestParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.PathSuffix == "" {
		return Result{Status: StatusError, Message: "path_suffix param is required"}
	}
	if params.Parameters != nil && !slices.Contains(
		[]string{"query", "body", "query_or_body"},
		params.Parameters.In,
	) {
		return Result{
			Status:  StatusError,
			Message: "parameters.in must be query, body or query_or_body",
		}
	}
	exchange, err := decodeOID4VCIExchange(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}

	var endpoint *url.URL
	if params.Endpoint != nil {
		metadata, ok := exchange.metadata[params.Endpoint.Metadata]
		if !ok {
			return Result{
				Status:  StatusInconclusive,
				Message: fmt.Sprintf("the exchange has no %s", params.Endpoint.Metadata),
			}
		}
		raw, _ := metadata[params.Endpoint.Field].(string)
		endpoint, err = url.Parse(raw)
		if raw == "" || err != nil || endpoint.Host == "" {
			return Result{
				Status: StatusInconclusive,
				Message: fmt.Sprintf(
					"%s does not publish a usable %s URL",
					params.Endpoint.Metadata,
					params.Endpoint.Field,
				),
			}
		}
	}

	requests := exchange.sessionRequestsEndingIn(params.PathSuffix)
	source := "recorded in the issuance session"
	if len(requests) == 0 && params.LedgerFallback {
		requests, source, err = exchange.ledgerRequestsAfterPAR(params.PathSuffix)
		if err != nil {
			return Result{Status: StatusInconclusive, Message: err.Error()}
		}
	}
	if len(requests) == 0 {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"no Wallet request ending in %s was recorded; Capture ties rejected requests to no session",
				params.PathSuffix,
			),
		}
	}

	for index, request := range requests {
		if problem := checkWalletRequest(request, params, endpoint); problem != "" {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"request %d of %d %s %s (%s): %s",
					index+1,
					len(requests),
					request.method,
					request.path,
					source,
					problem,
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"%d Wallet request(s) ending in %s, %s, pass every check",
			len(requests),
			params.PathSuffix,
			source,
		),
		Details: map[string]any{"requests": len(requests), "source": source},
	}
}

func checkWalletRequest(
	request capturedRequest,
	params walletRequestParams,
	endpoint *url.URL,
) string {
	if len(params.Methods) > 0 &&
		!slices.Contains(params.Methods, strings.ToUpper(request.method)) {
		return fmt.Sprintf("method %s is not one of %v", request.method, params.Methods)
	}
	proto, protoKnown := caseInsensitiveHeader(request.headers, "x-forwarded-proto")
	if params.RequireTLS {
		if !protoKnown {
			return "the capture does not record the request scheme, so TLS is not evidenced"
		}
		if !strings.EqualFold(proto, "https") {
			return fmt.Sprintf("the request arrived over %s, not TLS", proto)
		}
	}
	if endpoint != nil {
		if problem := checkRequestEndpoint(request, endpoint, proto); problem != "" {
			return problem
		}
	}
	if params.ForbidQuery && len(request.query) > 0 {
		keys := make([]string, 0, len(request.query))
		for key := range request.query {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return fmt.Sprintf("the request carries a query component (%s)", strings.Join(keys, ", "))
	}
	if params.MediaType != "" {
		mediaType, err := responseMediaType(request.headers)
		if err != nil {
			return err.Error()
		}
		if !strings.EqualFold(mediaType, params.MediaType) {
			return fmt.Sprintf(
				"Content-Type media type is %s, expected %s",
				mediaType,
				params.MediaType,
			)
		}
	}
	if params.Parameters != nil {
		if location := params.Parameters.In; location != "query" {
			if _, ok := request.body.(map[string]any); !ok || isRedactionMarker(request.body) {
				return fmt.Sprintf(
					"the request body is %s, so its parameters are not recorded",
					jsonTypeName(request.body),
				)
			}
		}
		members := requestParameters(request, params.Parameters.In)
		for _, name := range params.Parameters.Required {
			if !memberPresent(members, name) {
				return fmt.Sprintf(
					"required parameter %q is missing from the %s",
					name,
					params.Parameters.In,
				)
			}
		}
		for _, name := range params.Parameters.Absent {
			if memberPresent(members, name) {
				return fmt.Sprintf(
					"parameter %q must be absent from the %s",
					name,
					params.Parameters.In,
				)
			}
		}
	}
	return ""
}

// checkRequestEndpoint compares the request's host, port, path and query with
// the metadata endpoint. The host is the Host header the Wallet sent; a port
// left out on either side is the default port of the scheme.
func checkRequestEndpoint(request capturedRequest, endpoint *url.URL, proto string) string {
	host, ok := caseInsensitiveHeader(request.headers, "host")
	if !ok {
		return "the capture records no Host header"
	}
	requestHost, requestPort, err := net.SplitHostPort(host)
	if err != nil {
		requestHost, requestPort = host, ""
	}
	scheme := strings.ToLower(proto)
	if scheme == "" {
		scheme = endpoint.Scheme
	}
	if !strings.EqualFold(requestHost, endpoint.Hostname()) {
		return fmt.Sprintf("host %s is not the endpoint host %s", requestHost, endpoint.Hostname())
	}
	if defaultPort(requestPort, scheme) != defaultPort(endpoint.Port(), endpoint.Scheme) {
		return fmt.Sprintf(
			"port %s is not the endpoint port %s",
			defaultPort(requestPort, scheme),
			defaultPort(endpoint.Port(), endpoint.Scheme),
		)
	}
	endpointPath := endpoint.EscapedPath()
	if endpointPath == "" {
		endpointPath = "/"
	}
	if request.path != endpointPath && request.path != endpoint.Path {
		return fmt.Sprintf("path %s is not the endpoint path %s", request.path, endpointPath)
	}
	for key, values := range endpoint.Query() {
		value, _ := request.query[key].(string)
		if !slices.Contains(values, value) {
			return fmt.Sprintf(
				"the request omits the endpoint query member %s=%s",
				key,
				strings.Join(values, ","),
			)
		}
	}
	return ""
}

func defaultPort(port, scheme string) string {
	if port != "" {
		return port
	}
	switch strings.ToLower(scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

// configurationScopes maps each credential configuration of the issuer
// metadata to its scope.
func configurationScopes(issuerMetadata map[string]any) map[string]string {
	configurations, _ := issuerMetadata["credential_configurations_supported"].(map[string]any)
	scopes := make(map[string]string, len(configurations))
	for id, raw := range configurations {
		configuration, _ := raw.(map[string]any)
		if scope, _ := configuration["scope"].(string); scope != "" {
			scopes[id] = scope
		}
	}
	return scopes
}

// parScopes returns the space-separated values of the session PAR's scope.
func (e oid4vciExchange) parScopes() ([]string, error) {
	par, ok := e.pushedAuthorizationRequest()
	if !ok {
		return nil, fmt.Errorf("the session records no Pushed Authorization Request")
	}
	body, _ := par.body.(map[string]any)
	scope, ok := body["scope"].(string)
	if !ok {
		return nil, fmt.Errorf("the Pushed Authorization Request has no string scope parameter")
	}
	values := strings.Fields(scope)
	if len(values) == 0 {
		return nil, fmt.Errorf("the Pushed Authorization Request scope is empty")
	}
	return values, nil
}

// OID4VCIPARScopeSupportedValidator requires every space-separated value of the
// PAR's scope to be the scope of a credential configuration the issuer
// metadata advertises.
type OID4VCIPARScopeSupportedValidator struct{}

func (OID4VCIPARScopeSupportedValidator) ID() string { return "oid4vci.par_scope_supported" }

func (OID4VCIPARScopeSupportedValidator) Validate(_ context.Context, input Input) Result {
	exchange, err := decodeOID4VCIExchange(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	issuerMetadata, ok := exchange.metadata["issuer_metadata"]
	if !ok {
		return Result{Status: StatusInconclusive, Message: "the exchange has no issuer_metadata"}
	}
	values, err := exchange.parScopes()
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	advertised := map[string]bool{}
	for _, scope := range configurationScopes(issuerMetadata) {
		advertised[scope] = true
	}
	if len(advertised) == 0 {
		return Result{
			Status:  StatusInconclusive,
			Message: "the issuer metadata advertises no credential configuration scope",
		}
	}
	for _, value := range values {
		if !advertised[value] {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"scope value %q is not the scope of any advertised credential configuration",
					value,
				),
				Details: map[string]any{"scope": values},
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"every PAR scope value %v is an advertised credential configuration scope",
			values,
		),
		Details: map[string]any{"scope": values},
	}
}

// OID4VCICredentialConfigurationScopeRequestedValidator requires the Credential
// Request's credential_configuration_id to be a key of
// credential_configurations_supported whose scope the Wallet requested in its
// (pushed) Authorization Request.
type OID4VCICredentialConfigurationScopeRequestedValidator struct{}

func (OID4VCICredentialConfigurationScopeRequestedValidator) ID() string {
	return "oid4vci.credential_configuration_scope_requested"
}

func (OID4VCICredentialConfigurationScopeRequestedValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	exchange, err := decodeOID4VCIExchange(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	issuerMetadata, ok := exchange.metadata["issuer_metadata"]
	if !ok {
		return Result{Status: StatusInconclusive, Message: "the exchange has no issuer_metadata"}
	}
	raw, _ := exchange.session["raw"].(map[string]any)
	credentialRequest, ok := raw["credential_request"].(map[string]any)
	if !ok {
		return Result{Status: StatusFail, Message: "the session records no Credential Request"}
	}
	id, ok := credentialRequest["credential_configuration_id"].(string)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: "the Credential Request has no string credential_configuration_id",
		}
	}
	scope, advertised := configurationScopes(issuerMetadata)[id]
	if !advertised {
		configurations, _ := issuerMetadata["credential_configurations_supported"].(map[string]any)
		if _, known := configurations[id]; known {
			return Result{
				Status:  StatusInconclusive,
				Message: fmt.Sprintf("credential configuration %q advertises no scope", id),
			}
		}
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"credential_configuration_id %q is not a key of credential_configurations_supported",
				id,
			),
		}
	}
	values, err := exchange.parScopes()
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	if !slices.Contains(values, scope) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"credential configuration %q has scope %q, which the Authorization Request (scope %v) did not request",
				id,
				scope,
				values,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("credential configuration %q has the requested scope %q", id, scope),
	}
}

// keyAttestationMember is the JOSE header member Capture records as
// key_attestation_present instead of copying the attestation.
const keyAttestationMember = "key_attestation"

// OID4VCIProofJWTHeaderValidator checks the JOSE headers of the Wallet's `jwt`
// proofs from the session's raw.proof_headers.
type OID4VCIProofJWTHeaderValidator struct{}

func (OID4VCIProofJWTHeaderValidator) ID() string { return "oid4vci.proof_jwt_header" }

type proofJWTHeaderParams struct {
	RequiredMembers []string            `json:"required_members"`
	StringMembers   []string            `json:"string_members"`
	MemberEquals    map[string]string   `json:"member_equals"`
	MemberOneOf     map[string][]string `json:"member_one_of"`
}

// Validate requires every `jwt` proof header to carry required_members (the
// key_attestation member as recorded by key_attestation_present), to hold
// strings in string_members and the listed values in member_equals and
// member_one_of. Capture writes raw.proof_headers only after Credo accepted a
// proof, so an absent record is inconclusive. A Wallet that sent proofs of
// other types only is outside the `jwt` proof type the sources apply to.
func (OID4VCIProofJWTHeaderValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[proofJWTHeaderParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(
		params.RequiredMembers,
	)+len(
		params.StringMembers,
	)+len(
		params.MemberEquals,
	)+len(
		params.MemberOneOf,
	) == 0 {
		return Result{Status: StatusError, Message: "at least one header check param is required"}
	}
	entries, _ := input.Value.([]any)
	if len(entries) == 0 {
		return Result{
			Status: StatusInconclusive,
			Message: "Capture recorded no proof header; it records them only after Credo accepted the " +
				"proof, so the proof the Wallet sent is not evidenced",
		}
	}
	var headers []map[string]any
	proofTypes := map[string]bool{}
	for _, entry := range entries {
		header, ok := entry.(map[string]any)
		if !ok {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("proof header is %s, expected object", jsonTypeName(entry)),
			}
		}
		proofType, _ := header["proof_type"].(string)
		proofTypes[proofType] = true
		if proofType == "jwt" {
			headers = append(headers, header)
		}
	}
	if len(headers) == 0 {
		types := make([]string, 0, len(proofTypes))
		for proofType := range proofTypes {
			types = append(types, proofType)
		}
		sort.Strings(types)
		return Result{
			Status: StatusNotApplicable,
			Message: fmt.Sprintf(
				"the Wallet sent %s proofs only; the source applies to a Wallet that conveys a key attestation in a jwt proof",
				strings.Join(types, ", "),
			),
			Details: map[string]any{"proof_types": types},
		}
	}
	for index, header := range headers {
		if problem := checkProofJWTHeader(header, params); problem != "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("jwt proof %d of %d: %s", index+1, len(headers), problem),
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("%d jwt proof header(s) pass every check", len(headers)),
	}
}

func checkProofJWTHeader(header map[string]any, params proofJWTHeaderParams) string {
	for _, member := range params.RequiredMembers {
		if member == keyAttestationMember {
			if present, _ := header["key_attestation_present"].(bool); !present {
				return "the key_attestation header is absent"
			}
			continue
		}
		if !memberPresent(header, member) {
			return fmt.Sprintf("the %s header is absent", member)
		}
	}
	for _, member := range params.StringMembers {
		if _, ok := header[member].(string); !ok {
			return fmt.Sprintf(
				"the %s header is a JSON %s, expected string",
				member,
				jsonTypeName(header[member]),
			)
		}
	}
	for _, member := range sortedKeys(params.MemberEquals) {
		if value, _ := header[member].(string); value != params.MemberEquals[member] {
			return fmt.Sprintf(
				"the %s header is %v, expected %q",
				member,
				header[member],
				params.MemberEquals[member],
			)
		}
	}
	for _, member := range sortedKeys(params.MemberOneOf) {
		value, _ := header[member].(string)
		if !slices.Contains(params.MemberOneOf[member], value) {
			return fmt.Sprintf(
				"the %s header is %v, expected one of %v",
				member,
				header[member],
				params.MemberOneOf[member],
			)
		}
	}
	return ""
}

// OID4VCIIssuedCredentialPresentedValidator proves that a Wallet stored the
// credential it received in a valid Credential Response and can present it.
type OID4VCIIssuedCredentialPresentedValidator struct{}

func (OID4VCIIssuedCredentialPresentedValidator) ID() string {
	return "oid4vci.issued_credential_presented"
}

// Validate reads {issuance_session, presentation_session}: the issuance must
// have reached credential_issued, the Verifier must have validated the
// presentation, and one presented SD-JWT VC must come from the issuance
// session's issuer and be bound (cnf.jwk) to the holder key Capture verified at
// issuance. The key binding singles out the credential this issuance stored.
func (OID4VCIIssuedCredentialPresentedValidator) Validate(_ context.Context, input Input) Result {
	object, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %s, expected object", jsonTypeName(input.Value)),
		}
	}
	issuance, ok := normalizeJSONObject(object["issuance_session"])
	if !ok {
		return Result{Status: StatusFail, Message: "no issuance_session object"}
	}
	presentation, ok := normalizeJSONObject(object["presentation_session"])
	if !ok {
		return Result{Status: StatusFail, Message: "no presentation_session object"}
	}
	if status, _ := issuance["status"].(string); status != issuanceCompletedStatus {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"the issuance session is %q, so no Credential Response was delivered",
				status,
			),
		}
	}
	holderKeys := observedHolderKeys(issuance)
	if len(holderKeys) == 0 {
		return Result{
			Status:  StatusInconclusive,
			Message: "the issuance session records no verified holder key to identify the issued credential",
		}
	}
	checks, _ := presentation["checks"].(map[string]any)
	valid, _ := checks["presentation_valid"].(bool)
	if status, _ := presentation["status"].(string); status != "presentation_validated" || !valid {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"the Verifier session is %q with presentation_valid %v",
				status,
				checks["presentation_valid"],
			),
		}
	}
	issuer, _ := issuance["issuer_identifier"].(string)
	raw, _ := presentation["raw"].(map[string]any)
	byQuery, _ := raw["decoded_presentations"].(map[string]any)
	for _, queryID := range sortedKeys(byQuery) {
		presented, _ := byQuery[queryID].([]any)
		for _, entry := range presented {
			credential, _ := entry.(map[string]any)
			claims, _ := credential["claims"].(map[string]any)
			cnf, _ := claims["cnf"].(map[string]any)
			jwk, _ := cnf["jwk"].(map[string]any)
			if claims["iss"] != issuer || jwk == nil {
				continue
			}
			for _, key := range holderKeys {
				if sameECPublicKey(jwk, key) {
					return Result{
						Status: StatusPass,
						Message: fmt.Sprintf(
							"the Verifier validated credential query %q with the credential issued by %s to the holder key verified at issuance",
							queryID,
							issuer,
						),
					}
				}
			}
		}
	}
	return Result{
		Status:  StatusFail,
		Message: "no validated presentation carries the credential this issuance session issued (issuer and holder key)",
	}
}

func observedHolderKeys(issuance map[string]any) []map[string]any {
	observed, _ := issuance["observed"].(map[string]any)
	walletJWKS, _ := observed["wallet_jwks"].(map[string]any)
	jwks, _ := walletJWKS["jwks"].(map[string]any)
	entries, _ := jwks["keys"].([]any)
	keys := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if key, ok := entry.(map[string]any); ok {
			keys = append(keys, key)
		}
	}
	return keys
}

// sameECPublicKey compares the public members that identify a JWK.
func sameECPublicKey(a, b map[string]any) bool {
	for _, member := range []string{"kty", "crv", "x", "y", "n", "e"} {
		if !reflect.DeepEqual(a[member], b[member]) {
			return false
		}
	}
	return a["kty"] != nil
}
