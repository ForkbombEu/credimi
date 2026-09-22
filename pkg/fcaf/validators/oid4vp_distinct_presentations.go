// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

// OID4VPDistinctPresentationsValidator proves that the Wallet simultaneously
// holds several distinct credentials of one type. It is the precondition
// evidence the multiplicity cases need: counting presentations cannot
// distinguish two credentials from one credential presented twice, so the
// presentations are separated by a claim whose value differs per issued
// fixture. The evidence is either one captured exchange or an array of
// exchanges, which lets each fixture be proven through its own ordinary
// single-credential consent flow.
type OID4VPDistinctPresentationsValidator struct{}

// ID returns the validator identifier.
func (OID4VPDistinctPresentationsValidator) ID() string {
	return "oid4vp.distinct_presentations"
}

// Validate requires the credential queries selected by vct to return at least
// the requested number of presentations in total, each disclosing a different
// value of the discriminating claim.
func (OID4VPDistinctPresentationsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		VCT     string `json:"vct"`
		Claim   string `json:"claim"`
		Minimum int    `json:"minimum"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.VCT == "" {
		return Result{Status: StatusError, Message: "vct param is required"}
	}
	if params.Claim == "" {
		return Result{Status: StatusError, Message: "claim param is required"}
	}
	minimum := params.Minimum
	if minimum == 0 {
		minimum = 2
	}
	if minimum < 2 {
		return Result{Status: StatusError, Message: "minimum must be at least 2"}
	}

	exchanges, result := distinctPresentationExchanges(input.Value)
	if result != nil {
		return *result
	}

	seen := make(map[string]struct{}, minimum)
	for exchangeIndex, exchange := range exchanges {
		tokens, result := vctPresentations(exchange, params.VCT, exchangeIndex)
		if result != nil {
			return *result
		}
		for index, token := range tokens {
			presentation, err := evidence.ParseSDJWTPresentation(token)
			if err != nil {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"exchange %d presentation %d is not a valid SD-JWT: %v",
						exchangeIndex,
						index,
						err,
					),
				}
			}
			if presentation.Claims["vct"] != params.VCT {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"exchange %d presentation %d vct is %v, expected %q",
						exchangeIndex,
						index,
						presentation.Claims["vct"],
						params.VCT,
					),
				}
			}
			value, found := presentation.Claims[params.Claim]
			if !found {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"exchange %d presentation %d does not disclose %q",
						exchangeIndex,
						index,
						params.Claim,
					),
				}
			}
			seen[fmt.Sprintf("%v", value)] = struct{}{}
		}
	}

	if len(seen) < minimum {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"wallet presented %d distinct %s value(s), expected at least %d",
				len(seen),
				params.Claim,
				minimum,
			),
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"wallet holds %d credentials of type %q with distinct %s",
			len(seen),
			params.VCT,
			params.Claim,
		),
	}
}

// distinctPresentationExchanges accepts one captured exchange or an array of
// them.
func distinctPresentationExchanges(value any) ([]map[string]any, *Result) {
	if list, ok := value.([]any); ok {
		exchanges := make([]map[string]any, 0, len(list))
		for index, entry := range list {
			exchange, ok := normalizeJSONObject(entry)
			if !ok {
				return nil, &Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("DCQL evidence entry %d is not an object", index),
				}
			}
			exchanges = append(exchanges, exchange)
		}
		if len(exchanges) == 0 {
			return nil, &Result{Status: StatusFail, Message: "DCQL evidence array is empty"}
		}
		return exchanges, nil
	}
	exchange, ok := normalizeJSONObject(value)
	if !ok {
		return nil, &Result{Status: StatusFail, Message: "DCQL evidence is not an object"}
	}
	return []map[string]any{exchange}, nil
}

// vctPresentations returns the presentation strings one exchange returned for
// the credential query selecting the given vct.
func vctPresentations(root map[string]any, vct string, exchangeIndex int) ([]string, *Result) {
	queryValue, found := findObjectKey(root, "dcql_query")
	if !found {
		return nil, &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"exchange %d does not contain dcql_query",
				exchangeIndex,
			),
		}
	}
	query, ok := normalizeJSONObject(queryValue)
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("exchange %d dcql_query is not an object", exchangeIndex),
		}
	}
	credentialID, result := credentialQueryIDForVCT(query, vct)
	if result != nil {
		return nil, result
	}
	responseValue, found := findObjectKey(root, "vp_token")
	if !found {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("exchange %d does not contain vp_token", exchangeIndex),
		}
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("exchange %d vp_token is not an object", exchangeIndex),
		}
	}
	entries, ok := response[credentialID].([]any)
	if !ok || len(entries) == 0 {
		return nil, &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"exchange %d returned no presentation for credential query %q",
				exchangeIndex,
				credentialID,
			),
		}
	}
	tokens := make([]string, 0, len(entries))
	for index, entry := range entries {
		token, ok := entry.(string)
		if !ok || token == "" {
			return nil, &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"exchange %d presentation %d is not an SD-JWT",
					exchangeIndex,
					index,
				),
			}
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// credentialQueryIDForVCT returns the id of the credential query that selects
// the given vct.
func credentialQueryIDForVCT(query map[string]any, vct string) (string, *Result) {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return "", &Result{
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
					return "", &Result{
						Status:  StatusFail,
						Message: fmt.Sprintf("credential query for vct %q has no id", vct),
					}
				}
				return id, nil
			}
		}
	}
	return "", &Result{
		Status:  StatusFail,
		Message: fmt.Sprintf("dcql_query has no credential query for vct %q", vct),
	}
}
