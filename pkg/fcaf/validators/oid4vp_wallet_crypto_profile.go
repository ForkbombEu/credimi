// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
)

const (
	sha256HashName = "sha-256"
	// sessionNotObjectMessage reports evidence that is not a captured presentation session.
	sessionNotObjectMessage = "captured presentation session is not an object"
)

// ianaHashNames are the hash name strings of the IANA Named Information Hash
// Algorithm Registry. JWS algorithm identifiers such as "ES384" name signature
// algorithms, so they are deliberately not treated as hash algorithms.
var ianaHashNames = map[string]struct{}{
	"sha-224":     {},
	"sha-256":     {},
	"sha-384":     {},
	"sha-512":     {},
	"sha-512/224": {},
	"sha-512/256": {},
	"sha3-224":    {},
	"sha3-256":    {},
	"sha3-384":    {},
	"sha3-512":    {},
	"blake2s-256": {},
	"blake2b-256": {},
	"blake2b-512": {},
}

// hashAlgorithmNames collects every IANA hash name that appears as a string
// value or a member name anywhere inside a metadata value, lower-cased.
func hashAlgorithmNames(value any) []string {
	found := map[string]struct{}{}
	var walk func(any)
	record := func(text string) {
		name := strings.ToLower(strings.TrimSpace(text))
		if _, known := ianaHashNames[name]; known {
			found[name] = struct{}{}
		}
	}
	walk = func(current any) {
		switch typed := current.(type) {
		case string:
			record(typed)
		case []any:
			for _, item := range typed {
				walk(item)
			}
		case map[string]any:
			for key, item := range typed {
				record(key)
				walk(item)
			}
		}
	}
	walk(value)
	names := make([]string, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// presentedDigestAlgorithms returns the digest_algorithm Capture records for
// each decoded presentation, lower-cased, in query and index order.
func presentedDigestAlgorithms(session map[string]any) ([]string, *Result) {
	decoded, present := session["decoded_presentations"].(map[string]any)
	if !present {
		return nil, nil
	}
	queryIDs := make([]string, 0, len(decoded))
	for queryID := range decoded {
		queryIDs = append(queryIDs, queryID)
	}
	sort.Strings(queryIDs)
	algorithms := []string{}
	for _, queryID := range queryIDs {
		entries, ok := decoded[queryID].([]any)
		if !ok {
			return nil, &Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("decoded_presentations[%q] is not an array", queryID),
			}
		}
		for index, entry := range entries {
			presentation, _ := normalizeJSONObject(entry)
			algorithm := strings.ToLower(normalizeString(presentation["digest_algorithm"]))
			if algorithm == "" {
				return nil, &Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"decoded_presentations[%q][%d] records no digest_algorithm",
						queryID,
						index,
					),
				}
			}
			algorithms = append(algorithms, algorithm)
		}
	}
	return algorithms, nil
}

// postedWalletMetadata decodes the wallet_metadata a Wallet POSTed to the
// Request URI of a captured session.
func postedWalletMetadata(session map[string]any) (map[string]any, *Result) {
	observed, _ := normalizeJSONObject(session["observed"])
	requestURIPayload, _ := normalizeJSONObject(observed["request_uri_payload"])
	payload, ok := normalizeJSONObject(requestURIPayload["value"])
	if !ok {
		return nil, &Result{Status: StatusFail, Message: "Wallet did not POST to the Request URI"}
	}
	metadata, err := decodeWalletMetadata(payload["wallet_metadata"])
	if err != nil {
		return nil, &Result{Status: StatusFail, Message: err.Error()}
	}
	return metadata, nil
}

// OID4VPWalletMetadataHashAlgorithmsValidator applies HAIP Section 8: an
// entity using hash algorithms beyond SHA-256 SHOULD state them in its
// metadata. The session shows both halves: the Wallet's POSTed wallet_metadata
// and the digest algorithm of what it presented. Presenting a non-SHA-256
// credential is what proves the Wallet supports another hash algorithm; without
// it the source's profile applicability does not hold.
type OID4VPWalletMetadataHashAlgorithmsValidator struct{}

func (OID4VPWalletMetadataHashAlgorithmsValidator) ID() string {
	return "oid4vp.wallet_metadata_hash_algorithms"
}

func (OID4VPWalletMetadataHashAlgorithmsValidator) Validate(_ context.Context, input Input) Result {
	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: sessionNotObjectMessage}
	}
	metadata, result := postedWalletMetadata(session)
	if result != nil {
		return *result
	}
	presented, result := presentedDigestAlgorithms(session)
	if result != nil {
		return *result
	}
	supported := []string{}
	for _, algorithm := range presented {
		if algorithm != sha256HashName && !slices.Contains(supported, algorithm) {
			supported = append(supported, algorithm)
		}
	}
	if len(supported) == 0 {
		return Result{
			Status: StatusNotApplicable,
			Message: "wallet presented no credential digested with a hash algorithm other than " +
				"SHA-256, so its support for another one is not shown",
		}
	}
	declared := hashAlgorithmNames(metadata)
	for _, algorithm := range supported {
		if !slices.Contains(declared, algorithm) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"wallet presented a %s-digested credential but its wallet_metadata names no %s",
					algorithm,
					algorithm,
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"wallet_metadata names the supported hash algorithm(s) %s",
			strings.Join(supported, ", "),
		),
	}
}

// OID4VPPresentedDigestAlgorithmsValidator requires every presentation in a
// captured session to be digested with an allowed hash algorithm.
type OID4VPPresentedDigestAlgorithmsValidator struct{}

func (OID4VPPresentedDigestAlgorithmsValidator) ID() string {
	return "oid4vp.presented_digest_algorithms"
}

func (OID4VPPresentedDigestAlgorithmsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Allowed []string `json:"allowed"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(params.Allowed) == 0 {
		return Result{Status: StatusError, Message: "allowed param is required"}
	}
	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: sessionNotObjectMessage}
	}
	presented, result := presentedDigestAlgorithms(session)
	if result != nil {
		return *result
	}
	if len(presented) == 0 {
		return Result{Status: StatusFail, Message: "wallet returned no decoded presentation"}
	}
	for index, algorithm := range presented {
		if !slices.ContainsFunc(params.Allowed, func(allowed string) bool {
			return strings.EqualFold(allowed, algorithm)
		}) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"presentation %d is digested with %s, outside %s",
					index,
					algorithm,
					strings.Join(params.Allowed, ", "),
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"all %d presentation(s) are digested with %s",
			len(presented),
			strings.Join(params.Allowed, ", "),
		),
	}
}

// OID4VPClientMetadataHashAlgorithmsValidator reads the client_metadata signed
// into the delivered Request Object and fails when it names a hash algorithm
// outside the allowed list.
type OID4VPClientMetadataHashAlgorithmsValidator struct{}

func (OID4VPClientMetadataHashAlgorithmsValidator) ID() string {
	return "oid4vp.client_metadata_hash_algorithms"
}

func (OID4VPClientMetadataHashAlgorithmsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Allowed []string `json:"allowed"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(params.Allowed) == 0 {
		return Result{Status: StatusError, Message: "allowed param is required"}
	}
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	clientMetadata, ok := normalizeJSONObject(payload["client_metadata"])
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: "delivered Request Object carries no client_metadata object",
		}
	}
	for _, name := range hashAlgorithmNames(clientMetadata) {
		if !slices.ContainsFunc(params.Allowed, func(allowed string) bool {
			return strings.EqualFold(allowed, name)
		}) {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("delivered client_metadata names hash algorithm %s", name),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"delivered client_metadata names no hash algorithm other than %s",
			strings.Join(params.Allowed, ", "),
		),
	}
}

// OID4VPResponseEncUnsupportedErrorRequiredValidator decides a test whose
// profile is a Wallet supporting a single response content-encryption
// algorithm that the Verifier does not offer. The POSTed wallet_metadata
// states which enc values the Wallet supports: any value beyond the single one
// puts the Wallet outside the profile. Otherwise the Wallet must answer with
// the required error and no presentation.
type OID4VPResponseEncUnsupportedErrorRequiredValidator struct{}

func (OID4VPResponseEncUnsupportedErrorRequiredValidator) ID() string {
	return "oid4vp.response_enc_unsupported_error_required"
}

func (OID4VPResponseEncUnsupportedErrorRequiredValidator) Validate(
	ctx context.Context,
	input Input,
) Result {
	params, err := DecodeParams[struct {
		WalletOnlyEnc string `json:"wallet_only_enc"`
		Code          string `json:"code"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.WalletOnlyEnc == "" || params.Code == "" {
		return Result{Status: StatusError, Message: "wallet_only_enc and code params are required"}
	}
	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: sessionNotObjectMessage}
	}
	raw, _ := normalizeJSONObject(session["raw"])
	payload, err := compactJWTPart(raw["authorization_request_jwt"], 1)
	if err != nil {
		return Result{Status: StatusFail, Message: "served Request Object: " + err.Error()}
	}
	clientMetadata, _ := normalizeJSONObject(payload["client_metadata"])
	offered, _ := clientMetadata["encrypted_response_enc_values_supported"].([]any)
	if len(offered) == 0 {
		return Result{
			Status:  StatusFail,
			Message: "delivered client_metadata lists no encrypted_response_enc_values_supported",
		}
	}
	if slices.ContainsFunc(offered, func(value any) bool { return value == params.WalletOnlyEnc }) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"delivered client_metadata offers %s, so the precondition does not hold",
				params.WalletOnlyEnc,
			),
		}
	}

	metadata, result := postedWalletMetadata(session)
	if result != nil {
		return *result
	}
	supported, _ := metadata["authorization_encryption_enc_values_supported"].([]any)
	if len(supported) == 0 {
		return Result{
			Status:  StatusBlocked,
			Message: "wallet_metadata states no authorization_encryption_enc_values_supported",
		}
	}
	for _, value := range supported {
		if value != params.WalletOnlyEnc {
			return Result{
				Status: StatusNotApplicable,
				Message: fmt.Sprintf(
					"wallet_metadata also supports %v, so the Wallet does not support only %s",
					value,
					params.WalletOnlyEnc,
				),
			}
		}
	}
	return OID4VPErrorResponseRequiredValidator{}.Validate(ctx, Input{
		Value:  session,
		Params: map[string]any{"code": params.Code},
	})
}
