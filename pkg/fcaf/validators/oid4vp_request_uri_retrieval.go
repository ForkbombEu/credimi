// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// OID4VPRequestURIRetrievalValidator verifies the captured request_uri
// retrieval method and Host header. Capture Wallet attaches this record only
// after its session-specific /request endpoint has handled the request.
type OID4VPRequestURIRetrievalValidator struct{}

func (OID4VPRequestURIRetrievalValidator) ID() string {
	return "oid4vp.request_uri_retrieval"
}

func (OID4VPRequestURIRetrievalValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Method string `json:"method"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Method == "" {
		return Result{Status: StatusError, Message: "method param is required"}
	}

	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "presentation session is not an object"}
	}
	requestURI, ok := session["request_uri"].(string)
	if !ok || requestURI == "" {
		return Result{Status: StatusFail, Message: "presentation session request_uri is missing"}
	}
	parsedURI, err := url.ParseRequestURI(requestURI)
	if err != nil || parsedURI.Scheme != "https" || parsedURI.Host == "" || parsedURI.Path == "" {
		return Result{
			Status:  StatusFail,
			Message: "presentation session request_uri is not an absolute HTTPS URI with a path",
		}
	}

	raw, ok := normalizeJSONObject(session["raw"])
	if !ok {
		return Result{Status: StatusFail, Message: "presentation session raw evidence is missing"}
	}
	retrieval, ok := normalizeJSONObject(raw["request_uri_http"])
	if !ok {
		return Result{Status: StatusFail, Message: "request_uri HTTP retrieval evidence is missing"}
	}
	method, ok := retrieval["method"].(string)
	if !ok || method != params.Method {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"request_uri retrieval method is %q, expected %q",
				method,
				params.Method,
			),
		}
	}
	headers, ok := normalizeJSONObject(retrieval["headers"])
	if !ok {
		return Result{Status: StatusFail, Message: "request_uri retrieval headers are missing"}
	}
	host, ok := caseInsensitiveString(headers, "host")
	if !ok || host != parsedURI.Host {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"request_uri retrieval Host header is %q, expected %q",
				host,
				parsedURI.Host,
			),
		}
	}

	return Result{
		Status:  StatusPass,
		Message: "request_uri retrieval used the expected method and Host header at the session request endpoint",
	}
}

// OID4VPRequestURINotRetrievedValidator verifies that the Wallet did not
// retrieve the request object after rejecting an invalid request_uri_method.
type OID4VPRequestURINotRetrievedValidator struct{}

func (OID4VPRequestURINotRetrievedValidator) ID() string {
	return "oid4vp.request_uri_not_retrieved"
}

func (OID4VPRequestURINotRetrievedValidator) Validate(_ context.Context, input Input) Result {
	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "presentation session is not an object"}
	}
	if raw, ok := normalizeJSONObject(session["raw"]); ok {
		if _, retrieved := raw["request_uri_http"]; retrieved {
			return Result{
				Status:  StatusFail,
				Message: "request_uri HTTP retrieval evidence was recorded",
			}
		}
	}
	if events, ok := session["events"].([]any); ok {
		for _, event := range events {
			if event, ok := normalizeJSONObject(
				event,
			); ok &&
				event["type"] == "vp_request_retrieved" {
				return Result{
					Status:  StatusFail,
					Message: "request_uri retrieval event was recorded",
				}
			}
		}
	}
	return Result{Status: StatusPass, Message: "no request_uri retrieval was captured"}
}

func caseInsensitiveString(values map[string]any, key string) (string, bool) {
	for candidate, value := range values {
		if strings.EqualFold(candidate, key) {
			stringValue, ok := value.(string)
			return stringValue, ok
		}
	}
	return "", false
}
