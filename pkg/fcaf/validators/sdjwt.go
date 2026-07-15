// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/mail"
	"net/url"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/golang-jwt/jwt/v5"
)

type SDJWTClaimPresentValidator struct{}

func (SDJWTClaimPresentValidator) ID() string { return "sdjwt.claim_present" }

func (SDJWTClaimPresentValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	_, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", claim)}
	}
	return Result{Status: StatusPass, Message: fmt.Sprintf("claim %q is present", claim)}
}

type SDJWTClaimTypeValidator struct{}

func (SDJWTClaimTypeValidator) ID() string { return "sdjwt.claim_type" }

func (SDJWTClaimTypeValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim string `json:"claim"`
		Type  string `json:"type"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, params.Claim)
	if !ok {
		return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", params.Claim)}
	}
	if valueTypeName(value) != params.Type {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"claim %q is %s, expected %s",
				params.Claim,
				valueTypeName(value),
				params.Type,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("claim %q matches type %s", params.Claim, params.Type),
	}
}

type SDJWTClaimStringPrefixValidator struct{}

func (SDJWTClaimStringPrefixValidator) ID() string { return "sdjwt.claim_string_prefix" }

func (SDJWTClaimStringPrefixValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim  string `json:"claim"`
		Prefix string `json:"prefix"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Claim == "" {
		return Result{Status: StatusError, Message: "claim param is required"}
	}
	if params.Prefix == "" {
		return Result{Status: StatusError, Message: "prefix param is required"}
	}
	value, ok := sdjwtClaim(input.Value, params.Claim)
	if !ok {
		return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", params.Claim)}
	}
	text, ok := value.(string)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q is %T, expected string", params.Claim, value),
		}
	}
	if !strings.HasPrefix(text, params.Prefix) {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q does not start with %q", params.Claim, params.Prefix),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("claim %q starts with %q", params.Claim, params.Prefix),
	}
}

type SDJWTClaimUTF8StringValidator struct{}

func (SDJWTClaimUTF8StringValidator) ID() string { return "sdjwt.claim_utf8_string" }

func (SDJWTClaimUTF8StringValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", claim)}
	}
	text, ok := value.(string)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q is %T, expected string", claim, value),
		}
	}
	if !utf8.ValidString(text) {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q is not valid UTF-8", claim),
		}
	}
	if result := validateUTF8Vectors(input.Params); result != nil {
		return *result
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("claim %q is a valid UTF-8 string", claim),
	}
}

type SDJWTClaimRFC5322EmailValidator struct{}

func (SDJWTClaimRFC5322EmailValidator) ID() string { return "sdjwt.claim_rfc5322_email" }

func (SDJWTClaimRFC5322EmailValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", claim)}
	}
	text, ok := value.(string)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q is %T, expected string", claim, value),
		}
	}
	addr, err := mail.ParseAddress(text)
	if err != nil {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q is not a valid RFC 5322 address: %v", claim, err),
		}
	}
	if addr.Address != text || addr.Name != "" {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q must be a bare RFC 5322 addr-spec", claim),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("claim %q is a valid RFC 5322 address", claim),
	}
}

type PIDSDJWTVCTValidator struct{}

type SDJWTIssuerX509HeaderValidator struct{}

func (SDJWTIssuerX509HeaderValidator) ID() string { return "sdjwt.issuer_x509_header" }

func (SDJWTIssuerX509HeaderValidator) Validate(_ context.Context, input Input) Result {
	headers, ok := sdjwtProtectedHeaders(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "SD-JWT issuer protected headers are missing"}
	}
	rawChain, ok := headers["x5c"].([]any)
	if !ok || len(rawChain) == 0 {
		return Result{Status: StatusFail, Message: "SD-JWT issuer x5c chain is missing"}
	}
	for index, rawCertificate := range rawChain {
		encoded, ok := rawCertificate.(string)
		if !ok || encoded == "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("SD-JWT issuer x5c certificate %d is not a string", index),
			}
		}
		der, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"SD-JWT issuer x5c certificate %d is not base64 encoded",
					index,
				),
			}
		}
		if _, err := x509.ParseCertificate(der); err != nil {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("SD-JWT issuer x5c certificate %d is not valid DER", index),
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("SD-JWT issuer x5c chain contains %d certificate(s)", len(rawChain)),
	}
}

type SDJWTCNFConformsValidator struct{}

func (SDJWTCNFConformsValidator) ID() string { return "sdjwt.cnf_conforms" }

func (SDJWTCNFConformsValidator) Validate(_ context.Context, input Input) Result {
	presentations, ok := sdjwtPresentations(input.Value)
	if !ok || len(presentations) == 0 {
		return Result{Status: StatusFail, Message: "SD-JWT issuer payload is missing"}
	}
	for index, presentation := range presentations {
		result := validateSDJWTCNF(presentation)
		if result.Status != StatusPass {
			result.Message = fmt.Sprintf("presentation[%d]: %s", index, result.Message)
			return result
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"all %d SD-JWT cnf claims identify one proof-of-possession key",
			len(presentations),
		),
		Details: map[string]any{"presentation_count": len(presentations)},
	}
}

func validateSDJWTCNF(presentation *evidence.SDJWTPresentation) Result {
	payload := presentation.IssuerPayload
	rawCNF, ok := payload["cnf"]
	if !ok {
		return Result{Status: StatusFail, Message: `SD-JWT issuer claim "cnf" is missing`}
	}
	cnf, ok := rawCNF.(map[string]any)
	if !ok || len(cnf) == 0 {
		return Result{
			Status:  StatusFail,
			Message: `SD-JWT issuer claim "cnf" must be a non-empty object`,
		}
	}

	keyRepresentations := 0
	if rawJWK, exists := cnf["jwk"]; exists {
		keyRepresentations++
		jwk, ok := rawJWK.(map[string]any)
		if !ok {
			return Result{Status: StatusFail, Message: `cnf member "jwk" must be an object`}
		}
		if err := validatePublicJWK(jwk); err != nil {
			return Result{Status: StatusFail, Message: fmt.Sprintf("cnf JWK is invalid: %v", err)}
		}
	}
	if rawJWE, exists := cnf["jwe"]; exists {
		keyRepresentations++
		jwe, ok := rawJWE.(string)
		if !ok || len(strings.Split(jwe, ".")) != 5 {
			return Result{Status: StatusFail, Message: `cnf member "jwe" must be a compact JWE`}
		}
	}
	if rawJKU, exists := cnf["jku"]; exists {
		keyRepresentations++
		jku, ok := rawJKU.(string)
		if !ok {
			return Result{Status: StatusFail, Message: `cnf member "jku" must be a string`}
		}
		parsed, err := url.Parse(jku)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return Result{Status: StatusFail, Message: `cnf member "jku" must be an HTTPS URI`}
		}
	}
	if rawKID, exists := cnf["kid"]; exists {
		if _, hasJKU := cnf["jku"]; !hasJKU {
			keyRepresentations++
		}
		if text, ok := rawKID.(string); !ok || text == "" {
			return Result{
				Status:  StatusFail,
				Message: `cnf member "kid" must be a non-empty string`,
			}
		}
	}

	if keyRepresentations == 0 {
		return Result{Status: StatusFail, Message: "cnf has no supported confirmation method"}
	}
	if keyRepresentations > 1 {
		return Result{
			Status:  StatusFail,
			Message: "cnf identifies more than one proof-of-possession key",
		}
	}
	return Result{
		Status:  StatusPass,
		Message: "SD-JWT cnf claim identifies one proof-of-possession key",
	}
}

type SDJWTKeyBindingMatchesCNFValidator struct{}

func (SDJWTKeyBindingMatchesCNFValidator) ID() string { return "sdjwt.key_binding_matches_cnf" }

func (SDJWTKeyBindingMatchesCNFValidator) Validate(_ context.Context, input Input) Result {
	presentations, ok := sdjwtPresentations(input.Value)
	if !ok || len(presentations) == 0 {
		return Result{Status: StatusFail, Message: "SD-JWT presentation evidence is missing"}
	}
	algorithms := make([]string, 0, len(presentations))
	var blocked *Result
	for index, presentation := range presentations {
		result := validateSDJWTKeyBinding(presentation)
		if result.Status == StatusBlocked {
			result.Message = fmt.Sprintf("presentation[%d]: %s", index, result.Message)
			if blocked == nil {
				blocked = &result
			}
			continue
		}
		if result.Status != StatusPass {
			result.Message = fmt.Sprintf("presentation[%d]: %s", index, result.Message)
			return result
		}
		algorithm, _ := result.Details["alg"].(string)
		algorithms = append(algorithms, algorithm)
	}
	if blocked != nil {
		return *blocked
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"all %d KB-JWT signatures and sd_hash values are bound to their cnf.jwk and presented SD-JWT",
			len(presentations),
		),
		Details: map[string]any{
			"algorithms":          algorithms,
			"confirmation_method": "jwk",
			"presentation_count":  len(presentations),
		},
	}
}

func validateSDJWTKeyBinding(presentation *evidence.SDJWTPresentation) Result {
	cnf, ok := presentation.IssuerPayload["cnf"].(map[string]any)
	if !ok || len(cnf) == 0 {
		return Result{
			Status:  StatusFail,
			Message: `SD-JWT issuer claim "cnf" must be a non-empty object`,
		}
	}
	rawJWK, hasJWK := cnf["jwk"]
	if !hasJWK {
		return Result{
			Status:  StatusBlocked,
			Message: "cryptographic key-binding verification requires resolver evidence for cnf methods other than jwk",
		}
	}
	jwk, ok := rawJWK.(map[string]any)
	if !ok {
		return Result{Status: StatusFail, Message: `cnf member "jwk" must be an object`}
	}
	if presentation.KeyBindingJWT == "" || presentation.SDJWT == "" {
		return Result{Status: StatusFail, Message: "SD-JWT presentation does not contain a KB-JWT"}
	}

	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(
		presentation.KeyBindingJWT,
		claims,
		func(token *jwt.Token) (any, error) {
			return jwkVerificationKey(jwk, token.Method.Alg())
		},
		jwt.WithValidMethods(
			[]string{
				"ES256",
				"ES384",
				"ES512",
				"RS256",
				"RS384",
				"RS512",
				"PS256",
				"PS384",
				"PS512",
				"EdDSA",
			},
		),
		jwt.WithJSONNumber(),
		jwt.WithoutClaimsValidation(),
	)
	if err != nil || parsed == nil || !parsed.Valid {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("KB-JWT signature does not verify with cnf.jwk: %v", err),
		}
	}
	if typ, ok := parsed.Header["typ"].(string); !ok || typ != "kb+jwt" {
		return Result{
			Status:  StatusFail,
			Message: `KB-JWT protected header "typ" must equal "kb+jwt"`,
		}
	}
	if !validNumericDate(claims["iat"]) {
		return Result{Status: StatusFail, Message: `KB-JWT claim "iat" must be a valid NumericDate`}
	}
	if err := requireNonEmptyMapClaim(claims, "aud"); err != nil {
		return Result{Status: StatusFail, Message: `KB-JWT claim "aud" must be a non-empty string`}
	}
	if err := requireNonEmptyMapClaim(claims, "nonce"); err != nil {
		return Result{
			Status:  StatusFail,
			Message: `KB-JWT claim "nonce" must be a non-empty string`,
		}
	}
	rawSDHash, ok := claims["sd_hash"].(string)
	if !ok || rawSDHash == "" {
		return Result{
			Status:  StatusFail,
			Message: `KB-JWT claim "sd_hash" must be a non-empty string`,
		}
	}
	providedSDHash, err := base64.RawURLEncoding.DecodeString(rawSDHash)
	if err != nil || len(providedSDHash) != sha256.Size {
		return Result{
			Status:  StatusFail,
			Message: `KB-JWT claim "sd_hash" must be an unpadded base64url SHA-256 digest`,
		}
	}
	computedSDHash := sha256.Sum256([]byte(presentation.SDJWT))
	if subtle.ConstantTimeCompare(providedSDHash, computedSDHash[:]) != 1 {
		return Result{
			Status:  StatusFail,
			Message: "KB-JWT sd_hash does not bind the presented SD-JWT and disclosures",
		}
	}

	return Result{
		Status:  StatusPass,
		Message: "KB-JWT signature and sd_hash are bound to cnf.jwk and the presented SD-JWT",
		Details: map[string]any{"alg": parsed.Method.Alg(), "confirmation_method": "jwk"},
	}
}

func jwkVerificationKey(jwk map[string]any, algorithm string) (any, error) {
	if err := validatePublicJWK(jwk); err != nil {
		return nil, err
	}
	if declaredAlgorithm, ok := jwk["alg"].(string); ok && declaredAlgorithm != "" &&
		declaredAlgorithm != algorithm {
		return nil, fmt.Errorf(
			"JWK alg %q does not match KB-JWT alg %q",
			declaredAlgorithm,
			algorithm,
		)
	}
	if use, ok := jwk["use"].(string); ok && use != "" && use != "sig" {
		return nil, fmt.Errorf("JWK use %q does not permit signature verification", use)
	}
	if operations, exists := jwk["key_ops"]; exists && !jwkAllowsVerify(operations) {
		return nil, fmt.Errorf("JWK key_ops does not permit verification")
	}

	switch jwk["kty"] {
	case "EC":
		curveName, _ := jwk["crv"].(string)
		curve, expectedAlgorithm := elliptic.P256(), "ES256"
		switch curveName {
		case "P-384":
			curve, expectedAlgorithm = elliptic.P384(), "ES384"
		case "P-521":
			curve, expectedAlgorithm = elliptic.P521(), "ES512"
		}
		if algorithm != expectedAlgorithm {
			return nil, fmt.Errorf(
				"curve %s requires %s, got %s",
				curveName,
				expectedAlgorithm,
				algorithm,
			)
		}
		coordinateLength := (curve.Params().BitSize + 7) / 8
		x, _ := decodeBase64URLMember(jwk, "x", coordinateLength)
		y, _ := decodeBase64URLMember(jwk, "y", coordinateLength)
		return &ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(x),
			Y:     new(big.Int).SetBytes(y),
		}, nil
	case "RSA":
		if !strings.HasPrefix(algorithm, "RS") && !strings.HasPrefix(algorithm, "PS") {
			return nil, fmt.Errorf("RSA key cannot verify algorithm %s", algorithm)
		}
		modulus, _ := decodeBase64URLMember(jwk, "n", 0)
		exponentBytes, _ := decodeBase64URLMember(jwk, "e", 0)
		exponent := new(big.Int).SetBytes(exponentBytes)
		if !exponent.IsInt64() || exponent.Int64() <= 1 || exponent.Int64() > int64(^uint(0)>>1) {
			return nil, fmt.Errorf("JWK RSA exponent is invalid")
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: int(exponent.Int64())}, nil
	case "OKP":
		if jwk["crv"] != "Ed25519" || algorithm != "EdDSA" {
			return nil, fmt.Errorf(
				"only Ed25519 with EdDSA is supported for OKP signature verification",
			)
		}
		publicKey, _ := decodeBase64URLMember(jwk, "x", ed25519.PublicKeySize)
		return ed25519.PublicKey(publicKey), nil
	default:
		return nil, fmt.Errorf("unsupported JWK key type")
	}
}

func jwkAllowsVerify(value any) bool {
	operations, ok := value.([]any)
	if !ok || len(operations) == 0 {
		return false
	}
	for _, operation := range operations {
		if operation == "verify" {
			return true
		}
	}
	return false
}

func validNumericDate(value any) bool {
	var numeric float64
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return false
		}
		numeric = parsed
	case float64:
		numeric = typed
	default:
		return false
	}
	return numeric > 0 && !math.IsNaN(numeric) && !math.IsInf(numeric, 0)
}

func requireNonEmptyMapClaim(claims jwt.MapClaims, name string) error {
	value, ok := claims[name].(string)
	if !ok || value == "" {
		return fmt.Errorf("claim %q must be a non-empty string", name)
	}
	return nil
}

func validatePublicJWK(jwk map[string]any) error {
	if len(jwk) == 0 {
		return fmt.Errorf("JWK is empty")
	}
	for _, privateMember := range []string{"d", "p", "q", "dp", "dq", "qi", "oth", "k"} {
		if _, exists := jwk[privateMember]; exists {
			return fmt.Errorf("JWK contains private key member %q", privateMember)
		}
	}
	kty, ok := jwk["kty"].(string)
	if !ok || kty == "" {
		return fmt.Errorf("JWK kty must be a non-empty string")
	}
	switch kty {
	case "EC":
		crv, err := requiredString(jwk, "crv")
		if err != nil {
			return fmt.Errorf("JWK crv must be a non-empty string")
		}
		curve, ok := map[string]struct {
			bitSize int
			ecdh    ecdh.Curve
		}{
			"P-256": {bitSize: 256, ecdh: ecdh.P256()},
			"P-384": {bitSize: 384, ecdh: ecdh.P384()},
			"P-521": {bitSize: 521, ecdh: ecdh.P521()},
		}[crv]
		if !ok {
			return fmt.Errorf("unsupported EC curve %q", crv)
		}
		coordinateLength := (curve.bitSize + 7) / 8
		x, err := decodeBase64URLMember(jwk, "x", coordinateLength)
		if err != nil {
			return err
		}
		y, err := decodeBase64URLMember(jwk, "y", coordinateLength)
		if err != nil {
			return err
		}
		encodedPoint := make([]byte, 1+len(x)+len(y))
		encodedPoint[0] = 4
		copy(encodedPoint[1:], x)
		copy(encodedPoint[1+len(x):], y)
		if _, err := curve.ecdh.NewPublicKey(encodedPoint); err != nil {
			return fmt.Errorf("JWK EC coordinates are not on curve %q", crv)
		}
	case "RSA":
		if err := validateBase64URLMember(jwk, "n", 0); err != nil {
			return err
		}
		if err := validateBase64URLMember(jwk, "e", 0); err != nil {
			return err
		}
	case "OKP":
		crv, err := requiredString(jwk, "crv")
		if err != nil {
			return fmt.Errorf("JWK crv must be a non-empty string")
		}
		keyLength, ok := map[string]int{
			"Ed25519": 32,
			"Ed448":   57,
			"X25519":  32,
			"X448":    56,
		}[crv]
		if !ok {
			return fmt.Errorf("unsupported OKP curve %q", crv)
		}
		if err := validateBase64URLMember(jwk, "x", keyLength); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported public JWK kty %q", kty)
	}
	return nil
}

func validateBase64URLMember(jwk map[string]any, member string, expectedLength int) error {
	_, err := decodeBase64URLMember(jwk, member, expectedLength)
	return err
}

func decodeBase64URLMember(jwk map[string]any, member string, expectedLength int) ([]byte, error) {
	encoded, err := requiredString(jwk, member)
	if err != nil {
		return nil, fmt.Errorf("JWK %s must be a non-empty string", member)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(decoded) == 0 {
		return nil, fmt.Errorf("JWK %s must be unpadded base64url", member)
	}
	if expectedLength > 0 && len(decoded) != expectedLength {
		return nil, fmt.Errorf("JWK %s must decode to %d bytes", member, expectedLength)
	}
	return decoded, nil
}

func requiredString(object map[string]any, member string) (string, error) {
	text, ok := object[member].(string)
	if !ok || text == "" {
		return "", fmt.Errorf("member %q must be a non-empty string", member)
	}
	return text, nil
}

func sdjwtPresentation(value any) (*evidence.SDJWTPresentation, bool) {
	switch typed := value.(type) {
	case *evidence.SDJWTPresentation:
		return typed, typed != nil
	case evidence.SDJWTPresentation:
		return &typed, true
	default:
		return nil, false
	}
}

func sdjwtPresentations(value any) ([]*evidence.SDJWTPresentation, bool) {
	switch typed := value.(type) {
	case []*evidence.SDJWTPresentation:
		if len(typed) == 0 {
			return nil, false
		}
		for _, presentation := range typed {
			if presentation == nil {
				return nil, false
			}
		}
		return typed, true
	case []evidence.SDJWTPresentation:
		if len(typed) == 0 {
			return nil, false
		}
		presentations := make([]*evidence.SDJWTPresentation, len(typed))
		for index := range typed {
			presentations[index] = &typed[index]
		}
		return presentations, true
	default:
		presentation, ok := sdjwtPresentation(value)
		if !ok {
			return nil, false
		}
		return []*evidence.SDJWTPresentation{presentation}, true
	}
}

func sdjwtProtectedHeaders(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case *evidence.SDJWTPresentation:
		if typed == nil {
			return nil, false
		}
		return typed.ProtectedHeaders, true
	case evidence.SDJWTPresentation:
		return typed.ProtectedHeaders, true
	default:
		return nil, false
	}
}

const pidSDJWTVCTPrefix = "urn:eudi:pid:"

func (PIDSDJWTVCTValidator) ID() string { return "pid.sdjwt_vct_pid" }

func (PIDSDJWTVCTValidator) Validate(_ context.Context, input Input) Result {
	value, ok := sdjwtClaim(input.Value, "vct")
	if !ok {
		return Result{Status: StatusFail, Message: `claim "vct" is missing`}
	}
	text, ok := value.(string)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf(`claim "vct" is %T, expected string`, value),
		}
	}
	if !strings.HasPrefix(text, pidSDJWTVCTPrefix) {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf(`claim "vct" does not start with %q`, pidSDJWTVCTPrefix),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf(`claim "vct" starts with %q`, pidSDJWTVCTPrefix),
	}
}

type PIDSDJWTMandatoryClaimsValidator struct{}

func (PIDSDJWTMandatoryClaimsValidator) ID() string {
	return "pid.sdjwt_required_mandatory_claims_present"
}

func (PIDSDJWTMandatoryClaimsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		RequiredElements []string `json:"required_elements"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(params.RequiredElements) == 0 {
		return Result{
			Status:  StatusError,
			Message: "required_elements param is required",
		}
	}

	missing := []string{}
	for _, claim := range params.RequiredElements {
		if _, ok := claimFromValue(input.Value, claim); !ok {
			missing = append(missing, claim)
		}
	}
	if len(missing) > 0 {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("required mandatory PID SD-JWT claims are missing: %v", missing),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: "all required mandatory PID SD-JWT claims are present",
	}
}

func decodeClaimParam(params map[string]any) (string, error) {
	decoded, err := DecodeParams[struct {
		Claim string `json:"claim"`
	}](params)
	if err != nil {
		return "", err
	}
	if decoded.Claim == "" {
		return "", fmt.Errorf("claim param is required")
	}
	return decoded.Claim, nil
}

func sdjwtClaim(value any, claim string) (any, bool) {
	var claims map[string]any
	switch typed := value.(type) {
	case *evidence.SDJWTPresentation:
		if typed == nil {
			return nil, false
		}
		claims = typed.Claims
	case evidence.SDJWTPresentation:
		claims = typed.Claims
	case map[string]any:
		claims = typed
	default:
		return nil, false
	}
	return resolveObjectPath(claims, claim)
}

func claimFromValue(value any, claim string) (any, bool) {
	if got, ok := sdjwtClaim(value, claim); ok {
		return got, true
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	if namespaces, ok := root["namespaces"].(map[string]any); ok {
		for _, rawNamespace := range namespaces {
			namespace, ok := rawNamespace.(map[string]any)
			if !ok {
				continue
			}
			if got, ok := namespace[claim]; ok {
				return got, true
			}
		}
	}
	return nil, false
}

func valueTypeName(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case float64, float32, int, int64, int32, uint, uint64, uint32:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		if value == nil {
			return "null"
		}
		return reflect.TypeOf(value).String()
	}
}

func validateUTF8Vectors(params map[string]any) *Result {
	vectors, err := decodeVectors(params)
	if err != nil {
		return &Result{Status: StatusError, Message: err.Error()}
	}
	if len(vectors.Positive) == 0 && len(vectors.Negative) == 0 {
		return nil
	}

	for _, path := range vectors.Positive {
		file, err := loadVectorFile(path)
		if err != nil {
			return &Result{Status: StatusError, Message: err.Error()}
		}
		for _, tc := range file.Cases {
			data, err := vectorCaseBytes(tc)
			if err != nil {
				return &Result{Status: StatusError, Message: err.Error()}
			}
			if !utf8.Valid(data) {
				return &Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("positive UTF-8 vector %q is invalid", tc.ID),
				}
			}
		}
	}

	for _, path := range vectors.Negative {
		file, err := loadVectorFile(path)
		if err != nil {
			return &Result{Status: StatusError, Message: err.Error()}
		}
		for _, tc := range file.Cases {
			data, err := vectorCaseBytes(tc)
			if err != nil {
				return &Result{Status: StatusError, Message: err.Error()}
			}
			if utf8.Valid(data) {
				return &Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("negative UTF-8 vector %q is valid", tc.ID),
				}
			}
		}
	}

	return nil
}
