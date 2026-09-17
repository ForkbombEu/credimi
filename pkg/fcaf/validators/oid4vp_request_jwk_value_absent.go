// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPRequestJWKValueAbsentValidator verifies that no JWK published in an
// Authorization Request client_metadata uses a prohibited field value.
type OID4VPRequestJWKValueAbsentValidator struct{}

// ID returns the validator identifier.
func (OID4VPRequestJWKValueAbsentValidator) ID() string {
	return "oid4vp.request_jwk_value_absent"
}

// Validate rejects request metadata containing a JWK with the prohibited value.
func (OID4VPRequestJWKValueAbsentValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field string `json:"field"`
		Value any    `json:"value"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: fieldParamRequired}
	}
	if _, provided := input.Params["value"]; !provided {
		return Result{Status: StatusError, Message: "value param is required"}
	}

	requestObject, ok := input.Value.(string)
	if !ok || requestObject == "" {
		return Result{Status: StatusFail, Message: "signed request object is missing"}
	}
	metadata, err := requestClientMetadata(requestObject)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	jwks, ok := normalizeJSONObject(metadata["jwks"])
	if !ok {
		return Result{Status: StatusPass, Message: "client metadata does not publish JWKs"}
	}
	keys, ok := jwks["keys"].([]any)
	if !ok {
		return Result{Status: StatusFail, Message: "client metadata JWKs keys is not an array"}
	}
	for index, value := range keys {
		key, ok := normalizeJSONObject(value)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("client metadata JWK %d is not an object", index)}
		}
		if key[params.Field] == params.Value {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("client metadata JWK %d has prohibited %s value %v", index, params.Field, params.Value),
			}
		}
	}

	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("no client metadata JWK has %s value %v", params.Field, params.Value),
	}
}
