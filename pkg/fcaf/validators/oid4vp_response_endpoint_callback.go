// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// OID4VPResponseEndpointCallbackValidator verifies the HTTP response the
// verifier returns to the Wallet after a direct-post submission, captured in
// the session's raw.presentation_response_verifier_http record.
type OID4VPResponseEndpointCallbackValidator struct{}

func (OID4VPResponseEndpointCallbackValidator) ID() string {
	return "oid4vp.response_endpoint_callback"
}

func (OID4VPResponseEndpointCallbackValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Status                     int    `json:"status"`
		MediaType                  string `json:"media_type"`
		MatchConfiguredRedirectURI bool   `json:"match_configured_redirect_uri"`
		RequireResponseCode        bool   `json:"require_response_code"`
		RequireNoStore             bool   `json:"require_no_store"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Status == 0 && params.MediaType == "" && !params.MatchConfiguredRedirectURI &&
		!params.RequireResponseCode && !params.RequireNoStore {
		return Result{Status: StatusError, Message: "at least one callback check is required"}
	}

	evidence, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "callback evidence is not an object"}
	}
	response, ok := normalizeJSONObject(evidence["verifier_http"])
	if !ok {
		return Result{Status: StatusFail, Message: "verifier callback HTTP evidence is missing"}
	}

	if params.Status != 0 {
		status, ok := normalizeInteger(response["status"])
		if !ok {
			return Result{Status: StatusFail, Message: "verifier callback HTTP status is missing"}
		}
		if status != params.Status {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"verifier callback HTTP status is %d, expected %d",
					status,
					params.Status,
				),
			}
		}
	}

	headers, ok := normalizeJSONObject(response["headers"])
	if !ok {
		return Result{Status: StatusFail, Message: "verifier callback HTTP headers are missing"}
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
					"verifier callback media type is %q, expected %q",
					mediaType,
					params.MediaType,
				),
			}
		}
	}

	if params.RequireNoStore {
		cacheControl, ok := caseInsensitiveHeader(headers, "cache-control")
		if !ok || !hasCacheControlDirective(cacheControl, "no-store") {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"verifier callback Cache-Control is %q, expected no-store",
					cacheControl,
				),
			}
		}
	}

	if params.MatchConfiguredRedirectURI || params.RequireResponseCode {
		if err := validateCallbackRedirectURI(
			evidence,
			response,
			params.MatchConfiguredRedirectURI,
			params.RequireResponseCode,
		); err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
	}

	return Result{Status: StatusPass, Message: "verifier callback evidence matches"}
}

// validateCallbackRedirectURI checks the redirect URI the verifier handed to
// the Wallet in its callback body: that it targets the configured location and
// that it carries the response_code binding it to this Authorization Response.
func validateCallbackRedirectURI(
	evidence map[string]any,
	response map[string]any,
	matchConfigured bool,
	requireResponseCode bool,
) error {
	body, ok := response["body"].(string)
	if !ok {
		return fmt.Errorf("verifier callback body is missing")
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return fmt.Errorf("verifier callback body is not a JSON object")
	}
	returned, ok := payload["redirect_uri"].(string)
	if !ok || returned == "" {
		return fmt.Errorf("verifier callback body does not contain a redirect_uri string")
	}
	if matchConfigured {
		configured, ok := evidence["configured_redirect_uri"].(string)
		if !ok || configured == "" {
			return fmt.Errorf("configured redirect URI is missing")
		}
		if err := compareCallbackRedirectURI(configured, returned); err != nil {
			return err
		}
	}
	if requireResponseCode {
		return requireCallbackResponseCode(returned)
	}
	return nil
}

func normalizeInteger(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func hasCacheControlDirective(header string, directive string) bool {
	for _, part := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(part), directive) {
			return true
		}
	}
	return false
}

// compareCallbackRedirectURI requires the URI returned to the Wallet to target
// the same absolute HTTPS location as the configured redirect URI. The service
// appends its own response_code query parameter to the configured value, so
// query differences are not compared.
func compareCallbackRedirectURI(configured string, returned string) error {
	configuredURL, err := url.Parse(configured)
	if err != nil || configuredURL.Scheme == "" || configuredURL.Host == "" {
		return fmt.Errorf("configured redirect URI %q is not an absolute URI", configured)
	}
	returnedURL, err := url.Parse(returned)
	if err != nil || returnedURL.Scheme == "" || returnedURL.Host == "" {
		return fmt.Errorf("callback redirect_uri %q is not an absolute URI", returned)
	}
	if returnedURL.Scheme != "https" {
		return fmt.Errorf("callback redirect_uri scheme is %q, expected https", returnedURL.Scheme)
	}
	if !strings.EqualFold(configuredURL.Host, returnedURL.Host) ||
		configuredURL.Path != returnedURL.Path {
		return fmt.Errorf(
			"callback redirect_uri %q does not target the configured redirect URI %q",
			returned,
			configured,
		)
	}
	return nil
}

// requireCallbackResponseCode requires the URI handed to the Wallet to carry a
// non-empty response_code, which is what binds the redirect back to this
// Authorization Response. The value itself is generated by the verifier, so it
// is not compared against the session-creation value.
func requireCallbackResponseCode(returned string) error {
	returnedURL, err := url.Parse(returned)
	if err != nil {
		return fmt.Errorf("callback redirect_uri %q is not a URI", returned)
	}
	if returnedURL.Query().Get("response_code") == "" {
		return fmt.Errorf("callback redirect_uri %q carries no response_code", returned)
	}
	return nil
}
