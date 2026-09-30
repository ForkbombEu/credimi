// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

// OID4VPMDocDeviceResponsePerQueryValidator proves that a Wallet answering
// several ISO mdoc credential queries returned one separate DeviceResponse per
// query, each holding the credential that query selected. Every query must
// restrict at least one claim value, so the disclosed values show which stored
// mdoc answered it; counting DeviceResponses alone cannot tell three distinct
// credentials from one credential returned three times.
type OID4VPMDocDeviceResponsePerQueryValidator struct{}

// ID returns the validator identifier.
func (OID4VPMDocDeviceResponsePerQueryValidator) ID() string {
	return "oid4vp.mdoc_device_response_per_query"
}

// Validate reads the queries from the captured session, so the definition
// cannot drift from the request that was actually delivered.
func (OID4VPMDocDeviceResponsePerQueryValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	params, err := DecodeParams[struct {
		DocType    string `json:"doctype"`
		MinQueries int    `json:"min_queries"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.DocType == "" {
		return Result{Status: StatusError, Message: "doctype param is required"}
	}
	if params.MinQueries < 2 {
		return Result{Status: StatusError, Message: "min_queries must be at least 2"}
	}

	root, query, result := capturedDCQLQuery(input.Value)
	if result != nil {
		return *result
	}
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) < params.MinQueries {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"dcql_query holds %d credential queries, expected at least %d",
				len(credentials),
				params.MinQueries,
			),
		}
	}
	responseValue, found := findObjectKey(root, "vp_token")
	if !found {
		return Result{Status: StatusFail, Message: "wallet returned no vp_token"}
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return Result{Status: StatusFail, Message: "wallet vp_token is not an object"}
	}

	for index, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("credentials[%d] is not an object", index),
			}
		}
		if result := requireMDocQueryDeviceResponse(
			credential,
			response,
			params.DocType,
		); result != nil {
			return *result
		}
	}

	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"vp_token holds one %s DeviceResponse for each of the %d queries, "+
				"each satisfying its query's value restrictions",
			params.DocType,
			len(credentials),
		),
	}
}

// requireMDocQueryDeviceResponse checks that one credential query was answered
// by exactly one DeviceResponse whose document satisfies the query's values.
func requireMDocQueryDeviceResponse(
	credential map[string]any,
	response map[string]any,
	docType string,
) *Result {
	id, ok := credential["id"].(string)
	if !ok || id == "" {
		return &Result{Status: StatusFail, Message: "credential query has no id"}
	}
	meta, _ := normalizeJSONObject(credential["meta"])
	if credential["format"] != "mso_mdoc" || meta["doctype_value"] != docType {
		return &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("credential query %q does not request an %s mdoc", id, docType),
		}
	}
	restrictions, result := valueRestrictions(credential, id)
	if result != nil {
		return result
	}
	if len(restrictions) == 0 {
		return &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"credential query %q restricts no claim value, "+
					"so its answer cannot be told apart from another query's",
				id,
			),
		}
	}
	entries, _ := response[id].([]any)
	if len(entries) != 1 {
		return &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"vp_token[%q] holds %d presentations, expected one DeviceResponse",
				id,
				len(entries),
			),
		}
	}
	token, ok := entries[0].(string)
	if !ok || token == "" {
		return &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("vp_token[%q][0] is not an encoded DeviceResponse", id),
		}
	}
	presentation, err := evidence.ParseMDocPresentation(token)
	if err != nil {
		return &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("vp_token[%q][0] is not a valid DeviceResponse: %v", id, err),
		}
	}
	if len(presentation.Documents) != 1 || presentation.Documents[0].DocType != docType {
		return &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"vp_token[%q][0] DeviceResponse holds %d documents, expected one %s",
				id,
				len(presentation.Documents),
				docType,
			),
		}
	}
	for _, restriction := range restrictions {
		value, found := mdocRestrictedElement(presentation, restriction.path)
		if !found {
			return &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"vp_token[%q][0] does not disclose restricted element %v",
					id,
					restriction.path,
				),
			}
		}
		if !containsClaimValue(restriction.values, value) {
			return &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"vp_token[%q][0] discloses %v as %v, outside the query's values %v",
					id,
					restriction.path,
					value,
					restriction.values,
				),
			}
		}
	}
	return nil
}

// mdocRestrictedElement resolves a two-component mdoc claims path, namespace
// then element identifier, against the presented document.
func mdocRestrictedElement(presentation *evidence.MDocPresentation, path []any) (any, bool) {
	if len(path) != 2 {
		return nil, false
	}
	namespace, namespaceOK := path[0].(string)
	identifier, identifierOK := path[1].(string)
	if !namespaceOK || !identifierOK {
		return nil, false
	}
	element, found := presentation.Element(namespace, identifier)
	if !found {
		return nil, false
	}
	return element.Value, true
}
