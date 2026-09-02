// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"net/url"
)

// OID4VPClientIDMatchValidator verifies that client_id in the outer
// Authorization Request URI exactly matches the signed Request Object claim.
type OID4VPClientIDMatchValidator struct{}

func (OID4VPClientIDMatchValidator) ID() string {
	return "oid4vp.client_id_matches_request_object"
}

func (OID4VPClientIDMatchValidator) Validate(_ context.Context, input Input) Result {
	evidence, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "input is not an object"}
	}

	deeplink, ok := evidence["deeplink"].(string)
	if !ok || deeplink == "" {
		return Result{Status: StatusFail, Message: "deeplink is missing or not a string"}
	}
	outerClientID, err := clientIDFromDeeplink(deeplink)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}

	requestObject, ok := evidence["request_object"].(string)
	if !ok || requestObject == "" {
		return Result{Status: StatusFail, Message: "request_object is missing or not a string"}
	}
	payload, err := compactJWTPart(requestObject, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	requestObjectClientID, ok := payload["client_id"].(string)
	if !ok || requestObjectClientID == "" {
		return Result{Status: StatusFail, Message: "Request Object client_id is missing or not a string"}
	}
	if outerClientID != requestObjectClientID {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"outer client_id %q does not match Request Object client_id %q",
				outerClientID,
				requestObjectClientID,
			),
		}
	}

	return Result{
		Status:  StatusPass,
		Message: "outer and Request Object client_id values match exactly",
	}
}

func clientIDFromDeeplink(deeplink string) (string, error) {
	parsed, err := url.Parse(deeplink)
	if err != nil {
		return "", fmt.Errorf("parse deeplink: %w", err)
	}
	clientIDs, found := parsed.Query()["client_id"]
	if !found || len(clientIDs) != 1 || clientIDs[0] == "" {
		return "", fmt.Errorf("outer Authorization Request client_id is missing or invalid")
	}
	return clientIDs[0], nil
}
