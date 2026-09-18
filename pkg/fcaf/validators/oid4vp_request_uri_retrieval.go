// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

// OID4VPRequestURIRetrievalValidator verifies the captured request_uri
// retrieval method, HTTPS endpoint, and optional request headers and form body.
// Capture Wallet attaches this record only after its session-specific /request
// endpoint has handled the request.
type OID4VPRequestURIRetrievalValidator struct{}

// ID returns the validator identifier.
func (OID4VPRequestURIRetrievalValidator) ID() string {
	return "oid4vp.request_uri_retrieval"
}

// Validate checks the exact HTTP request that retrieved the request_uri.
func (OID4VPRequestURIRetrievalValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Method    string `json:"method"`
		MediaType string `json:"media_type"`
		Accept    string `json:"accept"`
		FormUTF8  bool   `json:"form_utf8"`
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
	if params.MediaType != "" {
		mediaType, err := responseMediaType(headers)
		if err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
		if mediaType != params.MediaType {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"request_uri retrieval media type is %q, expected %q",
					mediaType,
					params.MediaType,
				),
			}
		}
	}
	if params.Accept != "" {
		accept, ok := caseInsensitiveHeader(headers, "accept")
		if !ok || !strings.EqualFold(strings.TrimSpace(accept), params.Accept) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"request_uri retrieval Accept header is %q, expected %q",
					accept,
					params.Accept,
				),
			}
		}
	}
	if params.FormUTF8 {
		form, err := responseForm(retrieval)
		if err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
		for key, values := range form {
			if !utf8.ValidString(key) {
				return Result{
					Status:  StatusFail,
					Message: "request_uri form contains an invalid UTF-8 key",
				}
			}
			for _, value := range values {
				if !utf8.ValidString(value) {
					return Result{
						Status:  StatusFail,
						Message: "request_uri form contains an invalid UTF-8 value",
					}
				}
			}
		}
	}

	return Result{Status: StatusPass, Message: "request_uri retrieval HTTP evidence matches"}
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
