// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// verifierAttestationClientIDPrefix is the OpenID4VP Client Identifier Prefix
// that binds a Request Object to a Verifier Attestation
// (OpenID4VP 1.0 Section 5.9.3 / 12).
const verifierAttestationClientIDPrefix = "verifier_attestation:"

// OID4VPVerifierAttestationRequestValidator proves the bindings a delivered
// Request Object declares through the verifier_attestation: Client Identifier
// Prefix: the attestation carried in its jwt JOSE header, the attestation sub
// and iss, the Request Object signature under the attestation's cnf.jwk, the
// attestation signature under the published issuer JWKS, and the optional
// redirect_uris claim. FCAF tests change one binding at a time, so a verdict
// names the observed bindings instead of only accepting or rejecting the whole
// request.
type OID4VPVerifierAttestationRequestValidator struct{}

func (OID4VPVerifierAttestationRequestValidator) ID() string {
	return "oid4vp.verifier_attestation_request"
}

type verifierAttestationParams struct {
	JWTHeader            string `json:"jwt_header"`
	Subject              string `json:"subject"`
	Issuer               string `json:"issuer"`
	RequestSignature     string `json:"request_signature"`
	AttestationSignature string `json:"attestation_signature"`
	RedirectURIs         string `json:"redirect_uris"`
}

// validate rejects param sets that cannot name exactly one binding. The
// present-only params are rejected alongside jwt_header absent so a test cannot
// silently ask for contradicting expectations.
func (params verifierAttestationParams) validate() error {
	switch params.JWTHeader {
	case "absent":
		if params.Subject != "" || params.Issuer != "" ||
			params.RequestSignature != "" || params.AttestationSignature != "" ||
			params.RedirectURIs != "" {
			return errors.New("jwt_header absent takes no other params")
		}
		return nil
	case "present":
	default:
		return errors.New("jwt_header param must be present or absent")
	}
	if params.Subject != "client_id" && params.Subject != "other" {
		return errors.New("subject param must be client_id or other")
	}
	if params.Issuer == "" {
		return errors.New("issuer param is required")
	}
	if params.RequestSignature != "valid" && params.RequestSignature != "invalid" {
		return errors.New("request_signature param must be valid or invalid")
	}
	if params.AttestationSignature != "valid" && params.AttestationSignature != "invalid" {
		return errors.New("attestation_signature param must be valid or invalid")
	}
	switch params.RedirectURIs {
	case "absent", "matches_response_uri", "excludes_response_uri":
	default:
		return errors.New(
			"redirect_uris param must be absent, matches_response_uri or excludes_response_uri",
		)
	}
	return nil
}

func (OID4VPVerifierAttestationRequestValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[verifierAttestationParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if err := params.validate(); err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}

	request, failed := parseVerifierAttestationRequest(input.Value)
	if failed != nil {
		return *failed
	}
	if params.JWTHeader == "absent" {
		return verifyVerifierAttestationAbsent(request)
	}

	attestation, failed := parseVerifierAttestationToken(request)
	if failed != nil {
		return *failed
	}

	bindings := []string{"attestation in jwt header"}

	subject, failed := verifierAttestationSubjectBinding(
		params.Subject,
		attestation.subject,
		request.verifier,
	)
	if failed != nil {
		return *failed
	}
	bindings = append(bindings, subject)

	issuer, failed := verifierAttestationIssuerBinding(params.Issuer, attestation.issuer)
	if failed != nil {
		return *failed
	}
	bindings = append(bindings, issuer)

	requestSignature, failed := verifierAttestationRequestSignatureBinding(
		params.RequestSignature,
		request.raw,
		attestation.confirmedKey,
	)
	if failed != nil {
		return *failed
	}
	bindings = append(bindings, requestSignature)

	attestationSignature, failed := verifierAttestationSignatureBinding(
		params.AttestationSignature,
		attestation,
	)
	if failed != nil {
		return *failed
	}
	bindings = append(bindings, attestationSignature)

	redirect, failed := verifierAttestationRedirectBinding(
		params.RedirectURIs,
		request.claims,
		attestation.claims,
	)
	if failed != nil {
		return *failed
	}
	bindings = append(bindings, redirect)

	return Result{
		Status:  StatusPass,
		Message: "verifier_attestation request: " + strings.Join(bindings, ", "),
	}
}

// verifierAttestationRequest holds the delivered Request Object shape the
// checks need before any binding is evaluated.
type verifierAttestationRequest struct {
	raw      string
	header   map[string]any
	claims   jwt.MapClaims
	evidence map[string]any
	clientID string
	verifier string
}

// verifierAttestationToken holds the attestation delivered in the Request
// Object jwt header.
type verifierAttestationToken struct {
	raw          string
	header       map[string]any
	claims       map[string]any
	issuer       string
	subject      string
	confirmedKey *ecdsa.PublicKey
	issuerKeys   []any
}

// parseVerifierAttestationRequest decodes the input object and the compact JWS
// without verifying it, then extracts the verifier identifier from client_id.
func parseVerifierAttestationRequest(value any) (*verifierAttestationRequest, *Result) {
	evidence, ok := normalizeJSONObject(value)
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: "verifier_attestation request evidence is not an object",
		}
	}
	raw, ok := evidence["request_object"].(string)
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: "verifier_attestation Request Object is missing or not a compact JWS string",
		}
	}
	// Time claims are skipped: captured Request Objects are validated after
	// their short exp has passed.
	claims := jwt.MapClaims{}
	parsed, _, err := new(jwt.Parser).ParseUnverified(raw, claims)
	if err != nil {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("parse verifier_attestation Request Object: %v", err),
		}
	}
	clientID, _ := claims["client_id"].(string)
	verifier, prefixed := strings.CutPrefix(clientID, verifierAttestationClientIDPrefix)
	if !prefixed || verifier == "" {
		return nil, &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"Request Object client_id %q is not verifier_attestation:<verifier identifier>",
				clientID,
			),
		}
	}
	return &verifierAttestationRequest{
		raw:      raw,
		header:   parsed.Header,
		claims:   claims,
		evidence: evidence,
		clientID: clientID,
		verifier: verifier,
	}, nil
}

// verifyVerifierAttestationAbsent proves the Request Object carries no
// verifier attestation at all.
func verifyVerifierAttestationAbsent(request *verifierAttestationRequest) Result {
	if _, exists := request.header["jwt"]; exists {
		return Result{
			Status:  StatusFail,
			Message: "Request Object jwt header member carries a verifier attestation",
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"verifier_attestation request: no jwt header member, client_id %s",
			request.clientID,
		),
	}
}

// parseVerifierAttestationToken reads the jwt header attestation and the
// published issuer JWKS, rejecting shapes that cannot carry the declared
// bindings.
func parseVerifierAttestationToken(
	request *verifierAttestationRequest,
) (*verifierAttestationToken, *Result) {
	raw, ok := request.header["jwt"].(string)
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: "Request Object jwt header member is missing or not a compact JWS string",
		}
	}
	header, err := compactJWTPart(raw, 0)
	if err != nil {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("verifier attestation header: %v", err),
		}
	}
	claims, err := compactJWTPart(raw, 1)
	if err != nil {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("verifier attestation payload: %v", err),
		}
	}
	if typ, _ := header["typ"].(string); typ != "verifier-attestation+jwt" {
		return nil, &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"verifier attestation typ is %q, expected verifier-attestation+jwt",
				header["typ"],
			),
		}
	}
	issuer, ok := claims["iss"].(string)
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: "verifier attestation iss is not a string",
		}
	}
	subject, ok := claims["sub"].(string)
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: "verifier attestation sub is not a string",
		}
	}
	confirmedKey, failed := verifierAttestationConfirmedKey(claims)
	if failed != nil {
		return nil, failed
	}
	jwks, ok := normalizeJSONObject(request.evidence["attestation_issuer_jwks"])
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: "attestation_issuer_jwks is missing or not a JWKS object",
		}
	}
	issuerKeys, ok := jwks["keys"].([]any)
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: "attestation_issuer_jwks has no keys array",
		}
	}
	return &verifierAttestationToken{
		raw:          raw,
		header:       header,
		claims:       claims,
		issuer:       issuer,
		subject:      subject,
		confirmedKey: confirmedKey,
		issuerKeys:   issuerKeys,
	}, nil
}

// verifierAttestationConfirmedKey parses the cnf.jwk key the Request Object
// must be signed with.
func verifierAttestationConfirmedKey(claims map[string]any) (*ecdsa.PublicKey, *Result) {
	confirmation, ok := normalizeJSONObject(claims["cnf"])
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: "verifier attestation cnf is not an object",
		}
	}
	jwk, ok := normalizeJSONObject(confirmation["jwk"])
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: "verifier attestation cnf.jwk is not an object",
		}
	}
	key, err := p256PublicKeyFromJWK(jwk)
	if err != nil {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("verifier attestation cnf.jwk: %v", err),
		}
	}
	return key, nil
}

// verifierAttestationSubjectBinding compares the attestation sub with the
// Client Identifier verifier identifier.
func verifierAttestationSubjectBinding(
	expectation string,
	subject string,
	verifier string,
) (string, *Result) {
	switch expectation {
	case "client_id":
		if subject != verifier {
			return "", &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"verifier attestation sub %q does not match client_id %q",
					subject,
					verifier,
				),
			}
		}
		return fmt.Sprintf("sub matches client_id %s", verifier), nil
	case "other":
		if subject == verifier {
			return "", &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"verifier attestation sub %q matches client_id %q",
					subject,
					verifier,
				),
			}
		}
		return fmt.Sprintf("sub %s differs from client_id %s", subject, verifier), nil
	}
	return "", nil
}

// verifierAttestationIssuerBinding compares the attestation iss with the
// expected issuer.
func verifierAttestationIssuerBinding(expected, issuer string) (string, *Result) {
	if issuer != expected {
		return "", &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"verifier attestation iss %q does not match expected issuer %q",
				issuer,
				expected,
			),
		}
	}
	return fmt.Sprintf("iss %s", issuer), nil
}

// verifierAttestationRequestSignatureBinding proves the Request Object
// signature under the attestation cnf.jwk.
func verifierAttestationRequestSignatureBinding(
	expectation string,
	request string,
	key *ecdsa.PublicKey,
) (string, *Result) {
	valid, err := verifyVerifierAttestationES256(request, key)
	if err != nil {
		return "", &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("Request Object signature: %v", err),
		}
	}
	if valid != (expectation == "valid") {
		if valid {
			return "", &Result{
				Status:  StatusFail,
				Message: "Request Object signature verifies against the attestation cnf.jwk",
			}
		}
		return "", &Result{
			Status:  StatusFail,
			Message: "Request Object signature does not verify against the attestation cnf.jwk",
		}
	}
	if valid {
		return "Request Object signature verifies against cnf.jwk", nil
	}
	return "Request Object signature does not verify against cnf.jwk", nil
}

// verifierAttestationSignatureBinding proves the attestation signature under
// the issuer key its kid names.
func verifierAttestationSignatureBinding(
	expectation string,
	attestation *verifierAttestationToken,
) (string, *Result) {
	issuerKeyJWK, issuerKeyLabel, err := verifierAttestationIssuerKey(
		attestation.header,
		attestation.issuerKeys,
	)
	if err != nil {
		return "", &Result{Status: StatusFail, Message: err.Error()}
	}
	issuerKey, err := p256PublicKeyFromJWK(issuerKeyJWK)
	if err != nil {
		return "", &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("attestation issuer JWK: %v", err),
		}
	}
	valid, err := verifyVerifierAttestationES256(attestation.raw, issuerKey)
	if err != nil {
		return "", &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("verifier attestation signature: %v", err),
		}
	}
	if valid != (expectation == "valid") {
		if valid {
			return "", &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"verifier attestation signature verifies against %s",
					issuerKeyLabel,
				),
			}
		}
		return "", &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"verifier attestation signature does not verify against %s",
				issuerKeyLabel,
			),
		}
	}
	if valid {
		return fmt.Sprintf("attestation signature verifies against %s", issuerKeyLabel), nil
	}
	return fmt.Sprintf("attestation signature does not verify against %s", issuerKeyLabel), nil
}

// verifierAttestationRedirectBinding proves the optional redirect_uris claim
// against the Request Object response_uri.
func verifierAttestationRedirectBinding(
	expectation string,
	requestClaims map[string]any,
	attestationClaims map[string]any,
) (string, *Result) {
	if expectation == "absent" {
		if _, exists := attestationClaims["redirect_uris"]; exists {
			return "", &Result{
				Status:  StatusFail,
				Message: "verifier attestation carries redirect_uris",
			}
		}
		return "no redirect_uris", nil
	}
	responseURI, ok := requestClaims["response_uri"].(string)
	if !ok || responseURI == "" {
		return "", &Result{
			Status:  StatusFail,
			Message: "Request Object response_uri is missing or not a non-empty string",
		}
	}
	redirectURIs, ok := verifierAttestationStringArray(attestationClaims["redirect_uris"])
	if !ok {
		if expectation == "excludes_response_uri" {
			return "", &Result{
				Status:  StatusFail,
				Message: "verifier attestation redirect_uris is not a non-empty array of strings",
			}
		}
		return "", &Result{
			Status:  StatusFail,
			Message: "verifier attestation redirect_uris is not an array of strings",
		}
	}
	if expectation == "excludes_response_uri" && len(redirectURIs) == 0 {
		return "", &Result{
			Status:  StatusFail,
			Message: "verifier attestation redirect_uris is not a non-empty array of strings",
		}
	}
	contains := slices.Contains(redirectURIs, responseURI)
	if expectation == "matches_response_uri" {
		if !contains {
			return "", &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"verifier attestation redirect_uris %v does not contain response_uri %s",
					redirectURIs,
					responseURI,
				),
			}
		}
		return fmt.Sprintf("redirect_uris contains response_uri %s", responseURI), nil
	}
	if contains {
		return "", &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"verifier attestation redirect_uris %v contains response_uri %s",
				redirectURIs,
				responseURI,
			),
		}
	}
	return fmt.Sprintf("redirect_uris excludes response_uri %s", responseURI), nil
}

// verifyVerifierAttestationES256 reports whether the compact JWS signature
// verifies with key. A well-formed token whose signature does not verify
// returns (false, nil) so a negative expectation can be proven; a token that is
// not well-formed ES256 returns an error so an invalid expectation still needs
// a real signature mismatch.
func verifyVerifierAttestationES256(token string, key *ecdsa.PublicKey) (bool, error) {
	parsed, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return false, err
	}
	if alg, _ := parsed.Header["alg"].(string); alg != "ES256" {
		return false, fmt.Errorf("JWS alg is %q, expected ES256", parsed.Header["alg"])
	}
	_, err = jwt.Parse(
		token,
		func(*jwt.Token) (any, error) { return key, nil },
		jwt.WithValidMethods([]string{"ES256"}),
		jwt.WithoutClaimsValidation(),
	)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		return false, nil
	}
	return false, err
}

// verifierAttestationIssuerKey selects the issuer key the attestation header
// kid names. A kid the JWKS does not publish is never substituted, because an
// unattributable signature cannot be reported as either valid or invalid.
func verifierAttestationIssuerKey(
	header map[string]any,
	keys []any,
) (map[string]any, string, error) {
	rawKID, hasKID := header["kid"]
	if hasKID {
		kid, ok := rawKID.(string)
		if !ok {
			return nil, "", errors.New("verifier attestation kid is not a string")
		}
		for _, rawKey := range keys {
			key, ok := normalizeJSONObject(rawKey)
			if !ok {
				continue
			}
			if key["kid"] == kid {
				return key, "issuer key " + kid, nil
			}
		}
		return nil, "", fmt.Errorf(
			"verifier attestation kid %q is not published by attestation_issuer_jwks",
			kid,
		)
	}
	if len(keys) != 1 {
		return nil, "", fmt.Errorf(
			"verifier attestation has no kid and attestation_issuer_jwks publishes %d keys",
			len(keys),
		)
	}
	key, ok := normalizeJSONObject(keys[0])
	if !ok {
		return nil, "", errors.New("attestation_issuer_jwks key is not an object")
	}
	if kid, ok := key["kid"].(string); ok {
		return key, "issuer key " + kid, nil
	}
	return key, "the issuer JWKS key", nil
}

// verifierAttestationStringArray keeps redirect_uris an array of strings, so a
// non-string member can never satisfy or defeat a response_uri comparison.
func verifierAttestationStringArray(value any) ([]string, bool) {
	array, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(array))
	for _, item := range array {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, text)
	}
	return out, true
}
