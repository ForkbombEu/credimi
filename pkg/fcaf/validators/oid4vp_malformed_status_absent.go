// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

// Malformed status-reference shapes the capture issuer can emit.
const (
	statusShapeMissingStatusList = "missing_status_list"
	statusShapeNegativeIndex     = "negative_index"
	statusShapeMissingIndex      = "missing_index"
	statusShapeMalformedURI      = "malformed_uri"
	statusShapeMissingURI        = "missing_uri"
)

// OID4VPMalformedStatusCredentialAbsentValidator proves that a Wallet did not
// keep a Referenced Token whose status reference is malformed.
//
// Rejection cannot be observed directly, so it is probed: the query pins the
// document_number of the claim-set fixture the malformed token was issued on
// and asks for every match. A Wallet that rejected the token returns nothing.
// The shape check matters because one fixture may legitimately be issued more
// than once on the same device — the Wallet then holds a valid credential with
// the same document_number, and only the status claim distinguishes it from the
// one that had to be refused.
type OID4VPMalformedStatusCredentialAbsentValidator struct{}

// ID returns the validator identifier.
func (OID4VPMalformedStatusCredentialAbsentValidator) ID() string {
	return "oid4vp.malformed_status_credential_absent"
}

// Validate requires the pinned, multiple-enabled query to return no
// presentation carrying the malformed status shape.
func (OID4VPMalformedStatusCredentialAbsentValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	params, err := DecodeParams[struct {
		VCT   string `json:"vct"`
		Claim string `json:"claim"`
		Value any    `json:"value"`
		Shape string `json:"shape"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.VCT == "" {
		return Result{Status: StatusError, Message: "vct param is required"}
	}
	if params.Claim == "" {
		return Result{Status: StatusError, Message: claimParamRequired}
	}
	if params.Value == nil {
		return Result{Status: StatusError, Message: "value param is required"}
	}
	switch params.Shape {
	case statusShapeMissingStatusList,
		statusShapeNegativeIndex,
		statusShapeMissingIndex,
		statusShapeMalformedURI,
		statusShapeMissingURI:
	default:
		return Result{
			Status: StatusError,
			Message: "shape must be missing_status_list, negative_index, missing_index, " +
				"malformed_uri or missing_uri",
		}
	}

	root, query, result := capturedDCQLQuery(input.Value)
	if result != nil {
		return *result
	}
	credential, credentialID, result := credentialQueryForVCT(query, params.VCT)
	if result != nil {
		return *result
	}
	if multiple, ok := credential["multiple"].(bool); !ok || !multiple {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"credential query %q does not set multiple to true, "+
					"so a retained malformed credential could stay hidden",
				credentialID,
			),
		}
	}
	credentials := []any{credential}
	if result := requireClaimValueRestriction(
		credentials,
		[]any{params.Claim},
		params.Value,
	); result != nil {
		return *result
	}

	responseValue, found := findObjectKey(root, "vp_token")
	if !found {
		return Result{
			Status: StatusPass,
			Message: fmt.Sprintf(
				"wallet returned no presentation for %s %v",
				params.Claim,
				params.Value,
			),
		}
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return Result{Status: StatusFail, Message: "wallet vp_token is not an object"}
	}
	entries, _ := response[credentialID].([]any)
	if len(entries) == 0 {
		return Result{
			Status: StatusPass,
			Message: fmt.Sprintf(
				"wallet returned no presentation for %s %v",
				params.Claim,
				params.Value,
			),
		}
	}

	for index, entry := range entries {
		token, ok := entry.(string)
		if !ok || token == "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("vp_token[%q][%d] is not an SD-JWT", credentialID, index),
			}
		}
		presentation, err := evidence.ParseSDJWTPresentation(token)
		if err != nil {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"vp_token[%q][%d] is not a valid SD-JWT presentation: %v",
					credentialID,
					index,
					err,
				),
			}
		}
		if hasMalformedStatusShape(presentation.Claims, params.Shape) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"wallet retained a credential with %s %v whose status is %s",
					params.Claim,
					params.Value,
					params.Shape,
				),
			}
		}
	}

	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"none of the %d credential(s) with %s %v carries a %s status reference",
			len(entries),
			params.Claim,
			params.Value,
			params.Shape,
		),
	}
}

// hasMalformedStatusShape reports whether the presented claims exhibit the
// defect the issuer was asked to emit.
func hasMalformedStatusShape(claims map[string]any, shape string) bool {
	status, found := resolveObjectPath(claims, "status")
	if !found {
		return false
	}
	if _, ok := normalizeJSONObject(status); !ok {
		return false
	}
	statusList, listFound := resolveObjectPath(claims, "status.status_list")
	list, listIsObject := normalizeJSONObject(statusList)
	if shape == statusShapeMissingStatusList {
		return !listFound
	}
	if !listFound || !listIsObject {
		return false
	}

	switch shape {
	case statusShapeMissingIndex:
		_, indexFound := list["idx"]
		return !indexFound
	case statusShapeNegativeIndex:
		index, ok := claimPathIndex(list["idx"])
		return ok && index < 0
	case statusShapeMissingURI:
		_, uriFound := list["uri"]
		return !uriFound
	case statusShapeMalformedURI:
		text, ok := list["uri"].(string)
		if !ok {
			return true
		}
		parsed, err := url.Parse(text)
		return err != nil || !parsed.IsAbs() || parsed.Host == ""
	}
	return false
}
