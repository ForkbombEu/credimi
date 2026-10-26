// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// decodedCompactJWT is a compact JWS split into its signing input, decoded
// JOSE header and payload, and signature bytes.
type decodedCompactJWT struct {
	signingInput string
	header       map[string]any
	payload      map[string]any
	signature    []byte
}

func decodeCompactJWT(compact string) (decodedCompactJWT, error) {
	parts, err := compactParts(compact, 3)
	if err != nil {
		return decodedCompactJWT{}, err
	}
	header, err := decodeCompactObject(parts[0], "JOSE Header")
	if err != nil {
		return decodedCompactJWT{}, err
	}
	payload, err := decodeCompactObject(parts[1], "JWT payload")
	if err != nil {
		return decodedCompactJWT{}, err
	}
	signature, err := decodeCompactPart(parts[2], "signature")
	if err != nil {
		return decodedCompactJWT{}, err
	}
	return decodedCompactJWT{
		signingInput: parts[0] + "." + parts[1],
		header:       header,
		payload:      payload,
		signature:    signature,
	}, nil
}

// selectedJWTs decodes every selected value as a compact JWS.
func selectedJWTs(selected []selectedValue) ([]decodedCompactJWT, *Result) {
	tokens := make([]decodedCompactJWT, 0, len(selected))
	for _, item := range selected {
		compact, result := selectedString(item)
		if result != nil {
			return nil, result
		}
		token, err := decodeCompactJWT(compact)
		if err != nil {
			return nil, &Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("%s is not a compact JWS: %v", item.label, err),
			}
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// JOSEMemberValidator checks the members of a JWT's JOSE header or payload.
type JOSEMemberValidator struct{}

func (JOSEMemberValidator) ID() string { return "jose.member" }

type joseMemberParams struct {
	Part           string              `json:"part"`
	Required       []string            `json:"required"`
	String         []string            `json:"string"`
	Equals         map[string]string   `json:"equals"`
	OneOf          map[string][]string `json:"one_of"`
	EqualsExchange map[string]string   `json:"equals_exchange"`
}

// Validate requires, in params.part (header or payload) of every selected
// compact JWT: the members in required, string values in string, and the
// values in equals, one_of, and equals_exchange, which names an exchange value
// by path (e.g. aud = issuer_metadata.credential_issuer).
func (JOSEMemberValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[joseMemberParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Part != "header" && params.Part != "payload" {
		return Result{Status: StatusError, Message: "part must be header or payload"}
	}
	if len(params.Required)+len(params.String)+len(params.Equals)+len(params.OneOf)+
		len(params.EqualsExchange) == 0 {
		return Result{Status: StatusError, Message: "at least one member check is required"}
	}
	equals := map[string]string{}
	for member, value := range params.Equals {
		equals[member] = value
	}
	for member, path := range params.EqualsExchange {
		value, ok := exchangeValue(input.Value, path)
		text, isString := value.(string)
		if !ok || !isString {
			return Result{
				Status: StatusInconclusive,
				Message: fmt.Sprintf(
					"the exchange records no string %s to compare %s with",
					path,
					member,
				),
			}
		}
		equals[member] = text
	}
	selected, result := resolveSelection(input)
	if result != nil {
		return *result
	}
	tokens, result := selectedJWTs(selected)
	if result != nil {
		return *result
	}
	for index, token := range tokens {
		object := token.header
		if params.Part == "payload" {
			object = token.payload
		}
		if problem := checkJOSEMembers(object, params, equals); problem != "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("%s %s: %s", selected[index].label, params.Part, problem),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"the %s of %d JWT(s) passes every member check",
			params.Part,
			len(tokens),
		),
	}
}

func checkJOSEMembers(
	object map[string]any,
	params joseMemberParams,
	equals map[string]string,
) string {
	for _, member := range params.Required {
		if _, ok := object[member]; !ok {
			return fmt.Sprintf("member %s is absent", member)
		}
	}
	for _, member := range params.String {
		if _, ok := object[member].(string); !ok {
			return fmt.Sprintf(
				"member %s is a JSON %s, expected string",
				member,
				jsonTypeName(object[member]),
			)
		}
	}
	for _, member := range sortedKeys(equals) {
		if value, ok := object[member].(string); !ok || value != equals[member] {
			return fmt.Sprintf(
				"member %s is %v, expected %q",
				member,
				object[member],
				equals[member],
			)
		}
	}
	for _, member := range sortedKeys(params.OneOf) {
		if value, ok := object[member].(string); !ok ||
			!slices.Contains(params.OneOf[member], value) {
			return fmt.Sprintf(
				"member %s is %v, expected one of %v",
				member,
				object[member],
				params.OneOf[member],
			)
		}
	}
	return ""
}

// defaultSignatureAlgorithms is the intersection the device-binding sources
// require: on the ECCG agreed cryptographic mechanisms list, and one of ES256,
// ES384, ES512.
var defaultSignatureAlgorithms = []string{"ES256", "ES384", "ES512"}

// JOSESignatureAlgorithmValidator requires the alg of a JWT's protected header
// to be an accepted signature algorithm.
type JOSESignatureAlgorithmValidator struct{}

func (JOSESignatureAlgorithmValidator) ID() string { return "jose.signature_algorithm" }

func (JOSESignatureAlgorithmValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Allowed []string `json:"allowed"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	allowed := params.Allowed
	if len(allowed) == 0 {
		allowed = defaultSignatureAlgorithms
	}
	selected, result := resolveSelection(input)
	if result != nil {
		return *result
	}
	tokens, result := selectedJWTs(selected)
	if result != nil {
		return *result
	}
	algorithms := make([]string, 0, len(tokens))
	for index, token := range tokens {
		alg, _ := token.header["alg"].(string)
		if !slices.Contains(allowed, alg) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"%s is signed with alg %v, expected one of %s",
					selected[index].label,
					token.header["alg"],
					strings.Join(allowed, ", "),
				),
			}
		}
		algorithms = append(algorithms, alg)
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"signature algorithm(s) %s are accepted",
			strings.Join(algorithms, ", "),
		),
		Details: map[string]any{"alg": algorithms},
	}
}

// JOSESignatureValidator verifies the signature of a compact JWS with the
// public key the source names.
type JOSESignatureValidator struct{}

func (JOSESignatureValidator) ID() string { return "jose.signature_valid" }

// signatureKey names where the verification key comes from:
//
//   - x5c: the leaf certificate of the JWT's own x5c header;
//   - header_jwt: the claim at `claim` of the JWT carried in the JWT's own
//     header member `member` (the key_attestation of a jwt proof);
//   - sibling: the claim at `claim` of the JWT at `path` of the same request or
//     session the value was selected from (the Client Attestation of the PoP).
type signatureKey struct {
	Source string `json:"source"`
	Member string `json:"member"`
	Path   string `json:"path"`
	Claim  string `json:"claim"`
}

func (JOSESignatureValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Key signatureKey `json:"key"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	switch params.Key.Source {
	case "x5c":
	case "header_jwt":
		if params.Key.Member == "" || params.Key.Claim == "" {
			return Result{Status: StatusError, Message: "key.member and key.claim are required"}
		}
	case "sibling":
		if params.Key.Path == "" || params.Key.Claim == "" {
			return Result{Status: StatusError, Message: "key.path and key.claim are required"}
		}
	default:
		return Result{Status: StatusError, Message: "key.source must be x5c, header_jwt or sibling"}
	}
	selected, result := resolveSelection(input)
	if result != nil {
		return *result
	}
	tokens, result := selectedJWTs(selected)
	if result != nil {
		return *result
	}
	for index, token := range tokens {
		key, keySource, problem := signatureVerificationKey(token, selected[index], params.Key)
		if problem != nil {
			return *problem
		}
		alg, _ := token.header["alg"].(string)
		method := jwt.GetSigningMethod(alg)
		if method == nil || alg == "none" {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"%s alg %v is not a signature algorithm",
					selected[index].label,
					token.header["alg"],
				),
			}
		}
		if err := method.Verify(token.signingInput, token.signature, key); err != nil {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"the %s signature of %s does not verify with %s: %v",
					alg,
					selected[index].label,
					keySource,
					err,
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"%d signature(s) verify with the %s key",
			len(tokens),
			params.Key.Source,
		),
	}
}

func signatureVerificationKey(
	token decodedCompactJWT,
	item selectedValue,
	spec signatureKey,
) (any, string, *Result) {
	fail := func(status Status, format string, args ...any) (any, string, *Result) {
		return nil, "", &Result{
			Status:  status,
			Message: item.label + ": " + fmt.Sprintf(format, args...),
		}
	}
	switch spec.Source {
	case "x5c":
		chain, _ := token.header["x5c"].([]any)
		if len(chain) == 0 {
			return fail(StatusFail, "the JOSE Header has no x5c certificate")
		}
		encoded, _ := chain[0].(string)
		der, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return fail(StatusFail, "x5c[0] is not base64: %v", err)
		}
		certificate, err := x509.ParseCertificate(der)
		if err != nil {
			return fail(StatusFail, "x5c[0] is not an X.509 certificate: %v", err)
		}
		return certificate.PublicKey, "the x5c leaf " + certificate.Subject.String(), nil
	case "header_jwt":
		carried, ok := token.header[spec.Member].(string)
		if !ok {
			return fail(StatusFail, "the JOSE Header has no %s JWT", spec.Member)
		}
		return claimKey(carried, spec.Claim, "the "+spec.Member+" "+spec.Claim, fail)
	default:
		found := walkCapturePath(item.base, spec.Path, "")
		if len(found) == 0 {
			return fail(StatusFail, "the same capture records no %s", spec.Path)
		}
		carried, ok := found[0].value.(string)
		if !ok {
			return fail(
				StatusFail,
				"%s is a JSON %s, expected a JWT",
				spec.Path,
				jsonTypeName(found[0].value),
			)
		}
		return claimKey(carried, spec.Claim, spec.Path+" "+spec.Claim, fail)
	}
}

func claimKey(
	compact string,
	claim string,
	source string,
	fail func(Status, string, ...any) (any, string, *Result),
) (any, string, *Result) {
	carrier, err := decodeCompactJWT(compact)
	if err != nil {
		return fail(StatusFail, "the key-carrying JWT is not a compact JWS: %v", err)
	}
	found := walkCapturePath(carrier.payload, claim, "")
	if len(found) == 0 {
		return fail(StatusFail, "the key-carrying JWT has no %s claim", claim)
	}
	jwk, ok := found[0].value.(map[string]any)
	if !ok {
		return fail(
			StatusFail,
			"%s is a JSON %s, expected a JWK",
			claim,
			jsonTypeName(found[0].value),
		)
	}
	key, err := publicKeyFromJWK(jwk)
	if err != nil {
		return fail(StatusFail, "%s: %v", claim, err)
	}
	return key, source, nil
}

// publicKeyFromJWK parses an EC (P-256, P-384, P-521), RSA or Ed25519 public
// JWK.
func publicKeyFromJWK(jwk map[string]any) (any, error) {
	member := func(name string) ([]byte, error) {
		encoded, ok := jwk[name].(string)
		if !ok {
			return nil, fmt.Errorf("JWK has no %s", name)
		}
		return base64.RawURLEncoding.DecodeString(encoded)
	}
	switch jwk["kty"] {
	case "EC":
		curves := map[any]elliptic.Curve{
			"P-256": elliptic.P256(),
			"P-384": elliptic.P384(),
			"P-521": elliptic.P521(),
		}
		curve, ok := curves[jwk["crv"]]
		if !ok {
			return nil, fmt.Errorf("JWK curve %v is not supported", jwk["crv"])
		}
		x, err := member("x")
		if err != nil {
			return nil, err
		}
		y, err := member("y")
		if err != nil {
			return nil, err
		}
		size := (curve.Params().BitSize + 7) / 8
		if len(x) != size || len(y) != size {
			return nil, fmt.Errorf("JWK coordinates are not %d bytes", size)
		}
		encoded := append(append([]byte{4}, x...), y...)
		return ecdsa.ParseUncompressedPublicKey(curve, encoded)
	case "RSA":
		n, err := member("n")
		if err != nil {
			return nil, err
		}
		e, err := member("e")
		if err != nil {
			return nil, err
		}
		exponent := new(big.Int).SetBytes(e)
		if !exponent.IsInt64() || exponent.Int64() > 1<<31-1 {
			return nil, fmt.Errorf("JWK exponent is out of range")
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent.Int64())}, nil
	case "OKP":
		if jwk["crv"] != "Ed25519" {
			return nil, fmt.Errorf("JWK curve %v is not supported", jwk["crv"])
		}
		x, err := member("x")
		if err != nil {
			return nil, err
		}
		if len(x) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("JWK Ed25519 key is not %d bytes", ed25519.PublicKeySize)
		}
		return ed25519.PublicKey(x), nil
	default:
		return nil, fmt.Errorf("JWK kty %v is not supported", jwk["kty"])
	}
}
