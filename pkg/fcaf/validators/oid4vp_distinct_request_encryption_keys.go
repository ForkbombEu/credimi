// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"strings"
)

// OID4VPDistinctRequestEncryptionKeysValidator verifies that several
// Authorization Requests each published a different verifier encryption key.
// It makes a per-request key-agreement assertion meaningful: without distinct
// keys, a Wallet that reused one key for every response would still match each
// request's metadata.
type OID4VPDistinctRequestEncryptionKeysValidator struct{}

func (OID4VPDistinctRequestEncryptionKeysValidator) ID() string {
	return "oid4vp.distinct_request_encryption_keys"
}

func (OID4VPDistinctRequestEncryptionKeysValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	params, err := DecodeParams[struct {
		MinimumKeys int `json:"minimum_keys"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	minimum := params.MinimumKeys
	if minimum == 0 {
		minimum = 2
	}

	requestObjects, ok := input.Value.([]any)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %T, expected an array of request objects", input.Value),
		}
	}
	if len(requestObjects) < minimum {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"%d request object(s) captured, expected at least %d",
				len(requestObjects),
				minimum,
			),
		}
	}

	seen := make(map[string]int, len(requestObjects))
	for index, rawRequest := range requestObjects {
		requestObject, ok := rawRequest.(string)
		if !ok || requestObject == "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("request object %d is missing", index),
			}
		}
		metadata, err := requestClientMetadata(requestObject)
		if err != nil {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("request object %d: %v", index, err),
			}
		}
		key, err := encryptionJWK(metadata)
		if err != nil {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("request object %d: %v", index, err),
			}
		}
		identity := encryptionKeyIdentity(key)
		if previous, duplicated := seen[identity]; duplicated {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"request objects %d and %d published the same verifier encryption key",
					previous,
					index,
				),
			}
		}
		seen[identity] = index
	}

	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"%d request objects published distinct verifier encryption keys",
			len(requestObjects),
		),
	}
}

// encryptionKeyIdentity derives a comparison key from the JWK members that
// carry public key material, so an altered kid, use, or alg cannot disguise a
// reused key.
func encryptionKeyIdentity(key map[string]any) string {
	parts := make([]string, 0, 6)
	for _, member := range []string{"kty", "crv", "x", "y", "n", "e"} {
		value, _ := key[member].(string)
		parts = append(parts, member+"="+value)
	}
	return strings.Join(parts, "&")
}
