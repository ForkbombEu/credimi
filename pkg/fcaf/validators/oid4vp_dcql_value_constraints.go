// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

// capturedDCQLQuery decodes the evidence root and the DCQL query it carries.
// Every DCQL validator starts from this pair, so the three failure shapes are
// described once.
func capturedDCQLQuery(value any) (map[string]any, map[string]any, *Result) {
	root, ok := normalizeJSONObject(value)
	if !ok {
		return nil, nil, &Result{
			Status:  StatusFail,
			Message: "DCQL evidence is not an object",
		}
	}
	queryValue, found := findObjectKey(root, "dcql_query")
	if !found {
		return nil, nil, &Result{
			Status:  StatusFail,
			Message: "captured evidence does not contain dcql_query",
		}
	}
	query, ok := normalizeJSONObject(queryValue)
	if !ok {
		return nil, nil, &Result{
			Status:  StatusFail,
			Message: "captured dcql_query is not an object",
		}
	}
	return root, query, nil
}

// OID4VPDCQLValueConstraintsSatisfiedValidator proves that a Wallet applied the
// value restrictions of a DCQL query rather than merely answering it. Counting
// vp_token entries cannot decide this: several credentials of one type answer
// the same credential query, and only the disclosed claim values say which one
// came back. Every returned presentation must therefore disclose each
// restricted claim with a value drawn from that claim's values list, which
// fails as soon as one near-miss credential is released.
type OID4VPDCQLValueConstraintsSatisfiedValidator struct{}

// ID returns the validator identifier.
func (OID4VPDCQLValueConstraintsSatisfiedValidator) ID() string {
	return "oid4vp.dcql_value_constraints_satisfied"
}

// Validate reads the restrictions from the session-bound query itself, so the
// test definition cannot drift from the request that was actually delivered.
func (OID4VPDCQLValueConstraintsSatisfiedValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	params, err := DecodeParams[struct {
		VCT                string `json:"vct"`
		MinimumConstraints int    `json:"minimum_constraints"`
		RequireMultiple    bool   `json:"require_multiple"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.VCT == "" {
		return Result{Status: StatusError, Message: "vct param is required"}
	}
	minimum := params.MinimumConstraints
	if minimum == 0 {
		minimum = 1
	}
	if minimum < 1 {
		return Result{Status: StatusError, Message: "minimum_constraints must be positive"}
	}

	root, query, result := capturedDCQLQuery(input.Value)
	if result != nil {
		return *result
	}
	credential, credentialID, result := credentialQueryForVCT(query, params.VCT)
	if result != nil {
		return *result
	}
	if params.RequireMultiple {
		multiple, ok := credential["multiple"].(bool)
		if !ok || !multiple {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"credential query %q does not set multiple to true, "+
						"so a released near-miss credential could stay hidden",
					credentialID,
				),
			}
		}
	}

	restrictions, result := valueRestrictions(credential, credentialID)
	if result != nil {
		return *result
	}
	if len(restrictions) < minimum {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"credential query %q restricts %d claim value(s), expected at least %d",
				credentialID,
				len(restrictions),
				minimum,
			),
		}
	}

	presentations, result := vctPresentations(root, params.VCT, 0)
	if result != nil {
		return *result
	}
	for index, token := range presentations {
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
		for _, restriction := range restrictions {
			value, found := resolveClaimPath(presentation.Claims, restriction.path)
			if !found {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"released credential %d does not disclose restricted claim %v",
						index,
						restriction.path,
					),
				}
			}
			if !containsClaimValue(restriction.values, value) {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"released credential %d discloses %v as %v, "+
							"which the query excluded by restricting it to %v",
						index,
						restriction.path,
						value,
						restriction.values,
					),
				}
			}
		}
	}

	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"every released credential satisfies all %d value restriction(s) of query %q",
			len(restrictions),
			credentialID,
		),
	}
}

type dcqlValueRestriction struct {
	path   []any
	values []any
}

// valueRestrictions collects the requested claims that carry a values list.
func valueRestrictions(
	credential map[string]any,
	credentialID string,
) ([]dcqlValueRestriction, *Result) {
	claims, ok := credential["claims"].([]any)
	if !ok || len(claims) == 0 {
		return nil, &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"credential query %q does not request claims",
				credentialID,
			),
		}
	}
	restrictions := make([]dcqlValueRestriction, 0, len(claims))
	for index, rawClaim := range claims {
		claim, ok := normalizeJSONObject(rawClaim)
		if !ok {
			return nil, &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"credential query %q claims[%d] is not an object",
					credentialID,
					index,
				),
			}
		}
		values, ok := claim["values"].([]any)
		if !ok || len(values) == 0 {
			continue
		}
		path, ok := claim["path"].([]any)
		if !ok || len(path) == 0 {
			return nil, &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"credential query %q claims[%d] restricts values without a path",
					credentialID,
					index,
				),
			}
		}
		restrictions = append(restrictions, dcqlValueRestriction{path: path, values: values})
	}
	return restrictions, nil
}

// resolveClaimPath walks a DCQL claims path over decoded credential claims.
// String components select object members and integer components select array
// elements; the null selector is not supported because a value restriction
// applies to a single selected claim.
func resolveClaimPath(claims map[string]any, path []any) (any, bool) {
	var current any = claims
	for _, component := range path {
		switch typed := component.(type) {
		case string:
			object, ok := normalizeJSONObject(current)
			if !ok {
				return nil, false
			}
			current, ok = object[typed]
			if !ok {
				return nil, false
			}
		default:
			index, ok := claimPathIndex(component)
			if !ok {
				return nil, false
			}
			array, ok := current.([]any)
			if !ok || int(index) < 0 || int(index) >= len(array) {
				return nil, false
			}
			current = array[int(index)]
		}
	}
	return current, true
}

// containsClaimValue compares a disclosed value against a DCQL values list,
// normalizing numbers because YAML yields int and captured JSON yields float64.
func containsClaimValue(values []any, disclosed any) bool {
	for _, value := range values {
		if equalClaimPathComponent(disclosed, value) {
			return true
		}
	}
	return false
}

// credentialQueryForVCT returns the credential query selecting the given vct
// together with its id.
func credentialQueryForVCT(
	query map[string]any,
	vct string,
) (map[string]any, string, *Result) {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return nil, "", &Result{
			Status:  StatusFail,
			Message: "dcql_query does not contain credentials",
		}
	}
	for _, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			continue
		}
		meta, ok := normalizeJSONObject(credential["meta"])
		if !ok {
			continue
		}
		values, ok := meta["vct_values"].([]any)
		if !ok {
			continue
		}
		for _, rawValue := range values {
			if text, ok := rawValue.(string); ok && text == vct {
				id, ok := credential["id"].(string)
				if !ok || id == "" {
					return nil, "", &Result{
						Status:  StatusFail,
						Message: fmt.Sprintf("credential query for vct %q has no id", vct),
					}
				}
				return credential, id, nil
			}
		}
	}
	return nil, "", &Result{
		Status:  StatusFail,
		Message: fmt.Sprintf("dcql_query has no credential query for vct %q", vct),
	}
}
