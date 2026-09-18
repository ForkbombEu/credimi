// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPRequestEncryptionJWKValidator verifies the verifier encryption JWK that
// an Authorization Request published in client_metadata. It proves the
// precondition of a response-encryption negative test, for example a key that
// omits the alg parameter OpenID for Verifiable Presentations requires, or a key
// whose alg cannot be the JWE alg the Wallet would use.
type OID4VPRequestEncryptionJWKValidator struct{}

func (OID4VPRequestEncryptionJWKValidator) ID() string {
	return "oid4vp.request_encryption_jwk"
}

func (OID4VPRequestEncryptionJWKValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field   string `json:"field"`
		Value   any    `json:"value"`
		Present *bool  `json:"present"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: fieldParamRequired}
	}
	if params.Present == nil && params.Value == nil {
		return Result{Status: StatusError, Message: "value or present param is required"}
	}

	requestObject, ok := input.Value.(string)
	if !ok || requestObject == "" {
		return Result{Status: StatusFail, Message: "signed request object is missing"}
	}
	metadata, err := requestClientMetadata(requestObject)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	key, err := encryptionJWK(metadata)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}

	actual, found := key[params.Field]
	if params.Present != nil {
		if found != *params.Present {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"verifier encryption JWK field %q presence is %t, expected %t",
					params.Field,
					found,
					*params.Present,
				),
			}
		}
		if params.Value == nil {
			return Result{
				Status: StatusPass,
				Message: fmt.Sprintf(
					"verifier encryption JWK field %q presence matches",
					params.Field,
				),
			}
		}
	}
	if !found {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("verifier encryption JWK field %q is missing", params.Field),
		}
	}
	if actual != params.Value {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"verifier encryption JWK field %q is %v, expected %v",
				params.Field,
				actual,
				params.Value,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("verifier encryption JWK field %q matches", params.Field),
	}
}

// encryptionJWK returns the JWK a Wallet would select to encrypt the
// Authorization Response: the first key marked for encryption use, or the only
// published key when no key declares its use.
func encryptionJWK(metadata map[string]any) (map[string]any, error) {
	jwks, ok := normalizeJSONObject(metadata["jwks"])
	if !ok {
		return nil, fmt.Errorf("client metadata jwks is missing")
	}
	keys, ok := jwks["keys"].([]any)
	if !ok || len(keys) == 0 {
		return nil, fmt.Errorf("client metadata jwks contains no keys")
	}
	var fallback map[string]any
	for _, rawKey := range keys {
		key, ok := normalizeJSONObject(rawKey)
		if !ok {
			continue
		}
		if use, _ := key["use"].(string); use == "enc" {
			return key, nil
		}
		if fallback == nil {
			fallback = key
		}
	}
	if fallback == nil {
		return nil, fmt.Errorf("client metadata jwks contains no usable key")
	}
	return fallback, nil
}
