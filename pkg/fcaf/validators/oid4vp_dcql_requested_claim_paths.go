// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPDCQLRequestedClaimPathsValidator verifies that the Authorization
// Request captured on the session requested exactly the expected claims paths.
// Rejection cases need this: without it, a request for one unavailable claim
// and a request mixing an available claim with an unavailable one produce the
// same evidence, so one test's assertions cannot distinguish the other's
// precondition.
type OID4VPDCQLRequestedClaimPathsValidator struct{}

// ID returns the validator identifier.
func (OID4VPDCQLRequestedClaimPathsValidator) ID() string {
	return "oid4vp.dcql_requested_claim_paths"
}

// Validate requires the single captured credential query to carry exactly the
// expected claims paths, in any order and without extras.
func (OID4VPDCQLRequestedClaimPathsValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	params, err := DecodeParams[struct {
		Format string  `json:"format"`
		Paths  [][]any `json:"paths"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(params.Paths) == 0 {
		return Result{Status: StatusError, Message: "paths param is required"}
	}
	for index, path := range params.Paths {
		if len(path) == 0 {
			return Result{
				Status:  StatusError,
				Message: fmt.Sprintf("paths[%d] must not be empty", index),
			}
		}
	}

	_, query, result := capturedDCQLQuery(input.Value)
	if result != nil {
		return *result
	}
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) != 1 {
		return Result{
			Status:  StatusFail,
			Message: "dcql_query must contain exactly one credential query",
		}
	}
	credential, ok := normalizeJSONObject(credentials[0])
	if !ok {
		return Result{Status: StatusFail, Message: "credential query is not an object"}
	}
	if params.Format != "" && credential["format"] != params.Format {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"credential query format is %v, expected %q",
				credential["format"],
				params.Format,
			),
		}
	}
	claims, ok := credential["claims"].([]any)
	if !ok || len(claims) != len(params.Paths) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"credential query requests %d claims, expected %d",
				len(claims),
				len(params.Paths),
			),
		}
	}
	matched := make([]bool, len(params.Paths))
	for claimIndex, rawClaim := range claims {
		claim, ok := normalizeJSONObject(rawClaim)
		if !ok {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("claims[%d] is not an object", claimIndex),
			}
		}
		found := false
		for pathIndex, path := range params.Paths {
			if matched[pathIndex] {
				continue
			}
			if equalClaimPath(claim["path"], path) {
				matched[pathIndex] = true
				found = true
				break
			}
		}
		if !found {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"claims[%d].path %v is not an expected requested claims path",
					claimIndex,
					claim["path"],
				),
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: "authorization request requested exactly the expected claims paths",
	}
}
