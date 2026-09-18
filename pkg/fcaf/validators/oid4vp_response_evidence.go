// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"net/url"
	"reflect"
	"strings"
	"unicode/utf8"
)

// HTTPResponseMediaTypeValidator verifies the media type captured from an
// http-request activity result.
type HTTPResponseMediaTypeValidator struct{}

func (HTTPResponseMediaTypeValidator) ID() string { return "http.response_media_type" }

func (HTTPResponseMediaTypeValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		MediaType string `json:"media_type"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.MediaType == "" {
		return Result{Status: StatusError, Message: "media_type param is required"}
	}
	response, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "HTTP response evidence is not an object"}
	}
	headers, ok := normalizeJSONObject(response["headers"])
	if !ok {
		return Result{Status: StatusFail, Message: "HTTP response headers are missing"}
	}
	mediaType, err := responseMediaType(headers)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	if mediaType != params.MediaType {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"HTTP response media type is %q, expected %q",
				mediaType,
				params.MediaType,
			),
		}
	}
	return Result{Status: StatusPass, Message: "HTTP response media type matches"}
}

// OID4VPPresentationResponseHTTPValidator verifies the raw direct-post HTTP
// request captured on one presentation session.
type OID4VPPresentationResponseHTTPValidator struct{}

func (OID4VPPresentationResponseHTTPValidator) ID() string {
	return "oid4vp.presentation_response_http"
}

func (OID4VPPresentationResponseHTTPValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Method                   string `json:"method"`
		MediaType                string `json:"media_type"`
		RequireResponseParameter bool   `json:"require_response_parameter"`
		ResponseOnly             bool   `json:"response_only"`
		FormUTF8                 bool   `json:"form_utf8"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Method == "" && params.MediaType == "" && !params.RequireResponseParameter &&
		!params.ResponseOnly &&
		!params.FormUTF8 {
		return Result{Status: StatusError, Message: "at least one response HTTP check is required"}
	}
	response, err := presentationResponseHTTP(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	if params.Method != "" {
		method, _ := response["method"].(string)
		if method != params.Method {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"presentation response method is %q, expected %q",
					method,
					params.Method,
				),
			}
		}
	}
	headers, ok := normalizeJSONObject(response["headers"])
	if !ok {
		return Result{Status: StatusFail, Message: "presentation response headers are missing"}
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
					"presentation response media type is %q, expected %q",
					mediaType,
					params.MediaType,
				),
			}
		}
	}
	if params.RequireResponseParameter || params.ResponseOnly || params.FormUTF8 {
		form, err := responseForm(response)
		if err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
		if params.RequireResponseParameter {
			responses := form["response"]
			if len(responses) != 1 || responses[0] == "" {
				return Result{
					Status:  StatusFail,
					Message: "presentation response form must contain one non-empty response parameter",
				}
			}
		}
		if params.ResponseOnly && (len(form) != 1 || len(form["response"]) != 1) {
			return Result{
				Status:  StatusFail,
				Message: "presentation response form contains parameters other than response",
			}
		}
		if params.FormUTF8 {
			for key, values := range form {
				if !utf8.ValidString(key) {
					return Result{
						Status:  StatusFail,
						Message: "presentation response form contains an invalid UTF-8 key",
					}
				}
				for _, value := range values {
					if !utf8.ValidString(value) {
						return Result{
							Status:  StatusFail,
							Message: "presentation response form contains an invalid UTF-8 value",
						}
					}
				}
			}
		}
	}
	return Result{Status: StatusPass, Message: "presentation response HTTP evidence matches"}
}

// OID4VPResponseEncryptionValidator verifies response encryption properties
// from the original compact JWE captured in the direct-post form.
type OID4VPResponseEncryptionValidator struct{}

func (OID4VPResponseEncryptionValidator) ID() string { return "oid4vp.response_encryption" }

func (OID4VPResponseEncryptionValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[oid4vpResponseEncryptionParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if result := params.validate(); result != nil {
		return *result
	}
	captured, result := decodeResponseEncryptionEvidence(input.Value)
	if result != nil {
		return *result
	}
	for _, check := range []func(responseEncryptionEvidence) *Result{
		params.checkProtectedHeader,
		params.checkMetadataEnc,
		params.checkMetadataEncValues,
		params.checkGeneratedJWKs,
	} {
		if result := check(captured); result != nil {
			return *result
		}
	}
	return Result{Status: StatusPass, Message: "response encryption evidence matches"}
}

type oid4vpResponseEncryptionParams struct {
	MatchMetadataKID         bool     `json:"match_metadata_kid"`
	ExpectedEnc              string   `json:"expected_enc"`
	MetadataEnc              string   `json:"metadata_enc"`
	MetadataEncAbsent        bool     `json:"metadata_enc_absent"`
	MetadataEncValuesSupport []string `json:"metadata_enc_values_supported"`
	MetadataEncValuesOnly    bool     `json:"metadata_enc_values_exclusive"`
	PreserveGeneratedJWKs    bool     `json:"preserve_generated_jwks"`
}

func (p oid4vpResponseEncryptionParams) validate() *Result {
	if !p.MatchMetadataKID && p.ExpectedEnc == "" && p.MetadataEnc == "" &&
		!p.MetadataEncAbsent &&
		len(p.MetadataEncValuesSupport) == 0 &&
		!p.PreserveGeneratedJWKs {
		return &Result{
			Status:  StatusError,
			Message: "at least one response encryption check is required",
		}
	}
	if p.MetadataEnc != "" && p.MetadataEncAbsent {
		return &Result{
			Status:  StatusError,
			Message: "metadata_enc and metadata_enc_absent are contradictory",
		}
	}
	if p.MetadataEncValuesOnly && len(p.MetadataEncValuesSupport) == 0 {
		return &Result{
			Status:  StatusError,
			Message: "metadata_enc_values_exclusive requires metadata_enc_values_supported",
		}
	}
	return nil
}

// responseEncryptionEvidence holds the three artifacts every check needs: the
// verifier client metadata from the signed request, the protected header of the
// response JWE, and the probe's generated key set.
type responseEncryptionEvidence struct {
	metadata      map[string]any
	header        map[string]any
	generatedJWKs any
}

func decodeResponseEncryptionEvidence(
	value any,
) (responseEncryptionEvidence, *Result) {
	captured, ok := normalizeJSONObject(value)
	if !ok {
		return responseEncryptionEvidence{}, &Result{
			Status:  StatusFail,
			Message: "response encryption evidence is not an object",
		}
	}
	requestObject, ok := captured["request_object"].(string)
	if !ok || requestObject == "" {
		return responseEncryptionEvidence{}, &Result{
			Status:  StatusFail,
			Message: "signed request object is missing",
		}
	}
	metadata, err := requestClientMetadata(requestObject)
	if err != nil {
		return responseEncryptionEvidence{}, &Result{
			Status:  StatusFail,
			Message: err.Error(),
		}
	}
	response, ok := normalizeJSONObject(captured["presentation_response_http"])
	if !ok {
		return responseEncryptionEvidence{}, &Result{
			Status:  StatusFail,
			Message: "presentation response HTTP evidence is missing",
		}
	}
	form, err := responseForm(response)
	if err != nil {
		return responseEncryptionEvidence{}, &Result{
			Status:  StatusFail,
			Message: err.Error(),
		}
	}
	responses := form["response"]
	if len(responses) != 1 || responses[0] == "" {
		return responseEncryptionEvidence{}, &Result{
			Status:  StatusFail,
			Message: "presentation response form must contain one non-empty response parameter",
		}
	}
	header, err := compactJWEProtectedHeader(responses[0])
	if err != nil {
		return responseEncryptionEvidence{}, &Result{
			Status:  StatusFail,
			Message: err.Error(),
		}
	}
	return responseEncryptionEvidence{
		metadata:      metadata,
		header:        header,
		generatedJWKs: captured["generated_jwks"],
	}, nil
}

func (p oid4vpResponseEncryptionParams) checkProtectedHeader(
	captured responseEncryptionEvidence,
) *Result {
	if p.MatchMetadataKID {
		kid, _ := captured.header["kid"].(string)
		if kid == "" || !metadataContainsKID(captured.metadata, kid) {
			return &Result{
				Status:  StatusFail,
				Message: "JWE protected header kid does not match client metadata",
			}
		}
	}
	if p.ExpectedEnc != "" && captured.header["enc"] != p.ExpectedEnc {
		return &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"JWE protected header enc is %v, expected %q",
				captured.header["enc"],
				p.ExpectedEnc,
			),
		}
	}
	return nil
}

func (p oid4vpResponseEncryptionParams) checkMetadataEnc(
	captured responseEncryptionEvidence,
) *Result {
	metadataEnc, present := captured.metadata["authorization_encrypted_response_enc"]
	if p.MetadataEnc != "" && metadataEnc != p.MetadataEnc {
		return &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"client metadata authorization_encrypted_response_enc is %v, expected %q",
				metadataEnc,
				p.MetadataEnc,
			),
		}
	}
	if p.MetadataEncAbsent && present {
		return &Result{
			Status:  StatusFail,
			Message: "client metadata authorization_encrypted_response_enc is present",
		}
	}
	return nil
}

func (p oid4vpResponseEncryptionParams) checkMetadataEncValues(
	captured responseEncryptionEvidence,
) *Result {
	if len(p.MetadataEncValuesSupport) == 0 {
		return nil
	}
	advertised, ok := captured.metadata["encrypted_response_enc_values_supported"].([]any)
	if !ok {
		return &Result{
			Status:  StatusFail,
			Message: "client metadata encrypted_response_enc_values_supported is missing",
		}
	}
	supported := make(map[string]bool, len(advertised))
	for _, value := range advertised {
		if text, ok := value.(string); ok {
			supported[text] = true
		}
	}
	for _, required := range p.MetadataEncValuesSupport {
		if !supported[required] {
			return &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"client metadata encrypted_response_enc_values_supported is %v, expected to include %q",
					advertised,
					required,
				),
			}
		}
	}
	if p.MetadataEncValuesOnly && len(supported) != len(p.MetadataEncValuesSupport) {
		return &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"client metadata encrypted_response_enc_values_supported is %v, expected only %v",
				advertised,
				p.MetadataEncValuesSupport,
			),
		}
	}
	return nil
}

func (p oid4vpResponseEncryptionParams) checkGeneratedJWKs(
	captured responseEncryptionEvidence,
) *Result {
	if p.PreserveGeneratedJWKs &&
		!reflect.DeepEqual(captured.metadata["jwks"], captured.generatedJWKs) {
		return &Result{
			Status:  StatusFail,
			Message: "client metadata jwks differs from generated probe jwks",
		}
	}
	return nil
}

func presentationResponseHTTP(value any) (map[string]any, error) {
	session, ok := normalizeJSONObject(value)
	if !ok {
		return nil, fmt.Errorf("presentation session is not an object")
	}
	raw, ok := normalizeJSONObject(session["raw"])
	if !ok {
		return nil, fmt.Errorf("presentation session raw evidence is missing")
	}
	response, ok := normalizeJSONObject(raw["presentation_response_http"])
	if !ok {
		return nil, fmt.Errorf("presentation response HTTP evidence is missing")
	}
	return response, nil
}

func responseMediaType(headers map[string]any) (string, error) {
	contentType, ok := caseInsensitiveHeader(headers, "content-type")
	if !ok {
		return "", fmt.Errorf("Content-Type header is missing")
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", fmt.Errorf("parse Content-Type: %w", err)
	}
	return mediaType, nil
}

func caseInsensitiveHeader(headers map[string]any, key string) (string, bool) {
	for header, value := range headers {
		if !strings.EqualFold(header, key) {
			continue
		}
		switch typed := value.(type) {
		case string:
			return typed, typed != ""
		case []string:
			if len(typed) == 1 && typed[0] != "" {
				return typed[0], true
			}
		case []any:
			if len(typed) == 1 {
				if text, ok := typed[0].(string); ok && text != "" {
					return text, true
				}
			}
		}
	}
	return "", false
}

func responseForm(response map[string]any) (url.Values, error) {
	body, ok := response["body"].(string)
	if !ok {
		return nil, fmt.Errorf("presentation response body is missing")
	}
	form, err := url.ParseQuery(body)
	if err != nil {
		return nil, fmt.Errorf("parse presentation response form: %w", err)
	}
	return form, nil
}

func requestClientMetadata(requestObject string) (map[string]any, error) {
	parts := strings.Split(requestObject, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("signed request object is not a compact JWS")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("signed request object payload is not valid base64url")
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("signed request object payload is not valid JSON")
	}
	metadata, ok := normalizeJSONObject(claims["client_metadata"])
	if !ok {
		return nil, fmt.Errorf("signed request object client_metadata is missing")
	}
	return metadata, nil
}

func metadataContainsKID(metadata map[string]any, kid string) bool {
	jwks, ok := normalizeJSONObject(metadata["jwks"])
	if !ok {
		return false
	}
	keys, ok := jwks["keys"].([]any)
	if !ok {
		return false
	}
	for _, value := range keys {
		if key, ok := normalizeJSONObject(value); ok && key["kid"] == kid {
			return true
		}
	}
	return false
}

func compactJWEProtectedHeader(compact string) (map[string]any, error) {
	parts := strings.Split(compact, ".")
	if len(parts) != 5 {
		return nil, fmt.Errorf("input is not a compact JWE")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("JWE protected header is not valid base64url")
	}
	header := map[string]any{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("JWE protected header is not valid JSON")
	}
	return header, nil
}
