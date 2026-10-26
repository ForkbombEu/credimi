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
//     Wallet's requests Capture attributed to the session (in a session created
//     with evidence_mode "full", with header, proof and attestation values in
//     clear);
//   - session_requests: GET /sessions/{id}/requests, whose
//     unattributed_requests are the ledger requests with no session recorded
//     inside its window (metadata fetches, unused nonces, unmatched paths);
//   - session_creation: the POST /sessions response, with the delivered
//     deeplink;
//   - screenshots: the Maestro screenshot URLs of the Wallet step;
//   - issuer_metadata and authorization_server_metadata: the metadata the
//     Wallet resolved.
//
// Requests are located by path suffix and ordered by their `at` time, never by
// array position: Capture appends an attributed nonce request after the
// requests already recorded. Capture redacts sensitive values to
// {redacted, present, length}; only their presence is evidence.

// chainedAuthorizationServerPrefix is the path of Capture's internal
// auto-approving OAuth server. The issuer's authorization endpoint redirects the
// browser there, so its requests are Capture talking to itself, not the Wallet.
const chainedAuthorizationServerPrefix = "/authorization-servers/"

// capturedRequest is one HTTP request of a Capture capture.
type capturedRequest struct {
	at       time.Time
	id       string
	method   string
	path     string
	query    map[string]any
	body     any
	headers  map[string]any
	response map[string]any
	raw      map[string]any
}

// oid4vciExchange is a decoded Capture issuance exchange.
type oid4vciExchange struct {
	object       map[string]any
	session      map[string]any
	requests     []capturedRequest
	unattributed []capturedRequest
	window       *captureWindow
	metadata     map[string]map[string]any
}

// captureWindow is the interval GET /sessions/{id}/requests covers: from the
// session's creation to the time of the response.
type captureWindow struct {
	from time.Time
	to   time.Time
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
	exchange := oid4vciExchange{
		object:   object,
		session:  session,
		metadata: map[string]map[string]any{},
	}
	raw, _ := session["raw"].(map[string]any)
	exchange.requests = decodeCapturedRequests(raw["oid4vci_requests"])
	if listing, ok := normalizeJSONObject(object["session_requests"]); ok {
		exchange.unattributed = decodeCapturedRequests(listing["unattributed_requests"])
		window, _ := listing["window"].(map[string]any)
		from, fromErr := parseCaptureTime(window["from"])
		to, toErr := parseCaptureTime(window["to"])
		if fromErr == nil && toErr == nil {
			exchange.window = &captureWindow{from: from, to: to}
		}
	}
	for _, key := range []string{"issuer_metadata", "authorization_server_metadata"} {
		if metadata, ok := normalizeJSONObject(object[key]); ok {
			exchange.metadata[key] = metadata
		}
	}
	return exchange, nil
}

// decodeCapturedRequests decodes a list of captured requests ordered by `at`;
// requests with the same time keep their recorded order.
func decodeCapturedRequests(value any) []capturedRequest {
	entries, _ := value.([]any)
	requests := make([]capturedRequest, 0, len(entries))
	for _, entry := range entries {
		if request, ok := decodeCapturedRequest(entry); ok {
			requests = append(requests, request)
		}
	}
	sort.SliceStable(requests, func(i, j int) bool { return requests[i].at.Before(requests[j].at) })
	return requests
}

func decodeCapturedRequest(value any) (capturedRequest, bool) {
	object, ok := normalizeJSONObject(value)
	if !ok {
		return capturedRequest{}, false
	}
	request := capturedRequest{body: object["body"], raw: object}
	request.id, _ = object["id"].(string)
	request.method, _ = object["method"].(string)
	request.path, _ = object["path"].(string)
	request.query, _ = object["query"].(map[string]any)
	request.headers, _ = object["headers"].(map[string]any)
	request.response, _ = object["response"].(map[string]any)
	request.at, _ = parseCaptureTime(object["at"])
	return request, request.path != ""
}

// walletRequest reports whether a request path is a Wallet request ending in
// suffix, excluding Capture's chained OAuth server.
func walletRequest(path, suffix string) bool {
	return strings.HasSuffix(path, suffix) &&
		!strings.HasPrefix(path, chainedAuthorizationServerPrefix)
}

// sessionRequestsEndingIn returns the session's Wallet requests ending in
// suffix, ordered by time.
func (e oid4vciExchange) sessionRequestsEndingIn(suffix string) []capturedRequest {
	var matches []capturedRequest
	for _, request := range e.requests {
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

// issuerRejection returns the first Wallet request of the session the issuer
// answered with an error status. A request of a later flow step can then be
// missing because the issuer stopped the flow, as Credo does for a PAR that
// carries a JAR `request`, not because the Wallet failed to send it.
func (e oid4vciExchange) issuerRejection() (capturedRequest, bool) {
	for _, request := range e.requests {
		if strings.HasPrefix(request.path, chainedAuthorizationServerPrefix) {
			continue
		}
		if status, _ := request.response["status"].(float64); status >= 400 {
			return request, true
		}
	}
	return capturedRequest{}, false
}

// missingEvidence reports that the session lacks what a test inspects: blocked
// when the issuer rejected an earlier request of the flow, fail otherwise.
func (e oid4vciExchange) missingEvidence(message string) Result {
	rejected, ok := e.issuerRejection()
	if !ok {
		return Result{Status: StatusFail, Message: message}
	}
	reason := ""
	if body, _ := rejected.response["body"].(map[string]any); body != nil {
		if description, _ := body["error_description"].(string); description != "" {
			reason = ": " + description
		} else if code, _ := body["error"].(string); code != "" {
			reason = ": " + code
		}
	}
	return Result{
		Status: StatusBlocked,
		Message: fmt.Sprintf(
			"%s, because the issuer answered the %s %s request with %v%s, which stopped the flow",
			message,
			rejected.method,
			rejected.path,
			rejected.response["status"],
			reason,
		),
	}
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
	PathSuffix  string                   `json:"path_suffix"`
	Methods     []string                 `json:"methods"`
	Endpoint    *walletRequestEndpoint   `json:"endpoint"`
	RequireTLS  bool                     `json:"require_tls"`
	ForbidQuery bool                     `json:"forbid_query"`
	MediaType   string                   `json:"media_type"`
	Parameters  *walletRequestParameters `json:"parameters"`
}

// Validate locates every Wallet request whose path ends in params.path_suffix
// in the session and requires each to pass every configured check:
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
	if len(requests) == 0 {
		return exchange.missingEvidence(fmt.Sprintf(
			"the issuance session records no Wallet request ending in %s",
			params.PathSuffix,
		))
	}

	for index, request := range requests {
		if problem := checkWalletRequest(request, params, endpoint); problem != "" {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"request %d of %d %s %s: %s",
					index+1,
					len(requests),
					request.method,
					request.path,
					problem,
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"%d Wallet request(s) ending in %s pass every check",
			len(requests),
			params.PathSuffix,
		),
		Details: map[string]any{"requests": len(requests)},
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
	// The request as received (decrypted when the Wallet encrypted it): Capture
	// writes raw.credential_request only once Credo accepted the request.
	requests := exchange.sessionRequestsEndingIn("/credential")
	if len(requests) == 0 {
		return exchange.missingEvidence("the session records no Credential Request")
	}
	credentialRequest, ok := requests[len(requests)-1].body.(map[string]any)
	if !ok || isRedactionMarker(credentialRequest) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"the Credential Request body is %s, expected a JSON object",
				jsonTypeName(requests[len(requests)-1].body),
			),
		}
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

// OID4VCIIssuedCredentialPresentedValidator proves that a Wallet stored the
// credential it received in a valid Credential Response and can present it.
type OID4VCIIssuedCredentialPresentedValidator struct{}

func (OID4VCIIssuedCredentialPresentedValidator) ID() string {
	return "oid4vci.issued_credential_presented"
}

// Validate reads {issuance_session, presentation_session, screenshots}: the
// issuance must have reached credential_issued, the Verifier must have
// validated the presentation, and one presented SD-JWT VC must come from the
// issuance session's issuer and be bound (cnf.jwk) to the holder key Capture
// verified at issuance. The key binding singles out the credential this
// issuance stored. With params.informed_screenshot, the issuance step must also
// have stored the screenshot of the screen that informs the End-user.
func (OID4VCIIssuedCredentialPresentedValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		InformedScreenshot string `json:"informed_screenshot"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
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
	if status, _ := issuance["status"].(string); status != issuanceCompletedStatus {
		exchange, _ := decodeOID4VCIExchange(map[string]any{"capture_session": issuance})
		return exchange.missingEvidence(fmt.Sprintf(
			"the issuance session is %q, so no Credential Response was delivered",
			status,
		))
	}
	presentation, ok := normalizeJSONObject(object["presentation_session"])
	if !ok {
		return Result{Status: StatusFail, Message: "no presentation_session object"}
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
				if !sameECPublicKey(jwk, key) {
					continue
				}
				message := fmt.Sprintf(
					"the Verifier validated credential query %q with the credential issued by %s to the holder key verified at issuance",
					queryID,
					issuer,
				)
				if params.InformedScreenshot == "" {
					return Result{Status: StatusPass, Message: message}
				}
				names, failure := screenshotFileNames(object["screenshots"])
				if failure != nil {
					return Result{Status: StatusFail, Message: message + ", but " + failure.Message}
				}
				if !screenshotNamed(names, params.InformedScreenshot) {
					return Result{
						Status: StatusFail,
						Message: fmt.Sprintf(
							"%s, but the issuance step stored no %q screenshot informing the End-user (%s)",
							message,
							params.InformedScreenshot,
							strings.Join(names, ", "),
						),
					}
				}
				return Result{
					Status:  StatusPass,
					Message: message + "; the End-user was informed (" + params.InformedScreenshot + ")",
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
