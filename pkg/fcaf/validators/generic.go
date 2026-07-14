// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type EvidencePresentValidator struct{}

func (EvidencePresentValidator) ID() string {
	return "evidence.present"
}

type EvidenceNonEmptyValidator struct{}

func (EvidenceNonEmptyValidator) ID() string {
	return "evidence.non_empty"
}

type EvidenceMinimumItemsValidator struct{}

func (EvidenceMinimumItemsValidator) ID() string {
	return "evidence.minimum_items"
}

func (EvidenceMinimumItemsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		MinItems int `json:"min_items"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.MinItems < 1 {
		return Result{Status: StatusError, Message: "min_items must be greater than zero"}
	}

	itemCount := 0
	switch value := input.Value.(type) {
	case []any:
		itemCount = len(value)
	case []string:
		itemCount = len(value)
	default:
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("evidence value is %T, expected array", input.Value),
		}
	}
	if itemCount < params.MinItems {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"evidence contains %d item(s), expected at least %d",
				itemCount,
				params.MinItems,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("evidence contains at least %d item(s)", params.MinItems),
	}
}

func (EvidenceNonEmptyValidator) Validate(_ context.Context, input Input) Result {
	nonEmpty := false
	switch value := input.Value.(type) {
	case string:
		nonEmpty = value != ""
	case []any:
		nonEmpty = len(value) > 0
	case []string:
		nonEmpty = len(value) > 0
	case map[string]any:
		nonEmpty = len(value) > 0
	}
	if !nonEmpty {
		return Result{Status: StatusFail, Message: "evidence value is empty"}
	}
	return Result{Status: StatusPass, Message: "evidence value is non-empty"}
}

func (EvidencePresentValidator) Validate(_ context.Context, input Input) Result {
	if input.Value == nil {
		return Result{Status: StatusFail, Message: "evidence value is missing"}
	}
	return Result{Status: StatusPass, Message: "evidence value is present"}
}

type JSONFieldRequiredValidator struct{}

func (JSONFieldRequiredValidator) ID() string {
	return "json.field_required"
}

type JSONFieldEqualsValidator struct{}

func (JSONFieldEqualsValidator) ID() string {
	return "json.field_equals"
}

type JSONFieldPresenceValidator struct{}

func (JSONFieldPresenceValidator) ID() string {
	return "json.field_presence"
}

func (JSONFieldPresenceValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field   string `json:"field"`
		Present bool   `json:"present"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: "field param is required"}
	}
	if _, ok := input.Params["present"]; !ok {
		return Result{Status: StatusError, Message: "present param is required"}
	}
	obj, ok := input.Value.(map[string]any)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %T, expected object", input.Value),
		}
	}
	_, exists := obj[params.Field]
	if exists != params.Present {
		expectation := "absent"
		if params.Present {
			expectation = "present"
		}
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("field %q is not %s", params.Field, expectation),
		}
	}
	expectation := "absent"
	if params.Present {
		expectation = "present"
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("field %q is %s", params.Field, expectation),
	}
}

type JWTHeaderFieldEqualsValidator struct{}

func (JWTHeaderFieldEqualsValidator) ID() string {
	return "jwt.header_field_equals"
}

func (JWTHeaderFieldEqualsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field string `json:"field"`
		Value any    `json:"value"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: "field param is required"}
	}
	header, err := compactJWTPart(input.Value, 0)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	actual, exists := header[params.Field]
	if !exists {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("JWT header field %q is missing", params.Field),
		}
	}
	if !reflect.DeepEqual(actual, params.Value) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"JWT header field %q is %v, expected %v",
				params.Field,
				actual,
				params.Value,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("JWT header field %q equals %v", params.Field, params.Value),
	}
}

type JWTPayloadFieldEqualsValidator struct{}

func (JWTPayloadFieldEqualsValidator) ID() string {
	return "jwt.payload_field_equals"
}

func (JWTPayloadFieldEqualsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field string `json:"field"`
		Value any    `json:"value"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: "field param is required"}
	}
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	actual, exists := payload[params.Field]
	if !exists {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("JWT payload field %q is missing", params.Field),
		}
	}
	if !reflect.DeepEqual(actual, params.Value) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"JWT payload field %q is %v, expected %v",
				params.Field,
				actual,
				params.Value,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("JWT payload field %q equals %v", params.Field, params.Value),
	}
}

type JWTPayloadObjectKeysAllowedValidator struct{}

func (JWTPayloadObjectKeysAllowedValidator) ID() string {
	return "jwt.payload_object_keys_allowed"
}

type JWTPayloadFieldPresenceValidator struct{}

func (JWTPayloadFieldPresenceValidator) ID() string {
	return "jwt.payload_field_presence"
}

func (JWTPayloadFieldPresenceValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field   string `json:"field"`
		Present bool   `json:"present"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: "field param is required"}
	}
	if _, ok := input.Params["present"]; !ok {
		return Result{Status: StatusError, Message: "present param is required"}
	}
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	_, exists := payload[params.Field]
	if exists != params.Present {
		expectation := "absent"
		if params.Present {
			expectation = "present"
		}
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("JWT payload field %q is not %s", params.Field, expectation),
		}
	}
	expectation := "absent"
	if params.Present {
		expectation = "present"
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("JWT payload field %q is %s", params.Field, expectation),
	}
}

func (JWTPayloadObjectKeysAllowedValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field       string   `json:"field"`
		AllowedKeys []string `json:"allowed_keys"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: "field param is required"}
	}
	if len(params.AllowedKeys) == 0 {
		return Result{Status: StatusError, Message: "allowed_keys param is required"}
	}
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	value, exists := payload[params.Field]
	if !exists {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("JWT payload field %q is missing", params.Field),
		}
	}
	object, ok := value.(map[string]any)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("JWT payload field %q is not an object", params.Field),
		}
	}
	allowed := make(map[string]struct{}, len(params.AllowedKeys))
	for _, key := range params.AllowedKeys {
		allowed[key] = struct{}{}
	}
	for key := range object {
		if _, ok := allowed[key]; !ok {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"JWT payload field %q contains undefined key %q",
					params.Field,
					key,
				),
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("JWT payload field %q contains only allowed keys", params.Field),
	}
}

func compactJWTPart(value any, index int) (map[string]any, error) {
	token, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("input is %T, expected compact JWT", value)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("input is not a compact JWT")
	}
	partJSON, err := base64.RawURLEncoding.DecodeString(parts[index])
	if err != nil {
		return nil, fmt.Errorf("JWT part %d is not base64url encoded", index)
	}
	var part map[string]any
	if err := json.Unmarshal(partJSON, &part); err != nil {
		return nil, fmt.Errorf("JWT part %d is not a JSON object", index)
	}
	return part, nil
}

func (JSONFieldEqualsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field string `json:"field"`
		Value any    `json:"value"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: "field param is required"}
	}
	obj, ok := input.Value.(map[string]any)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %T, expected object", input.Value),
		}
	}
	actual, exists := obj[params.Field]
	if !exists {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("field %q is missing", params.Field),
		}
	}
	if !reflect.DeepEqual(actual, params.Value) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"field %q is %v, expected %v",
				params.Field,
				actual,
				params.Value,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("field %q equals %v", params.Field, params.Value),
	}
}

func (JSONFieldRequiredValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field string `json:"field"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: "field param is required"}
	}
	obj, ok := input.Value.(map[string]any)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %T, expected object", input.Value),
		}
	}
	if _, ok := obj[params.Field]; !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("required field %q is missing", params.Field),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("required field %q is present", params.Field),
	}
}

type MDocNamespaceElementPresentValidator struct{}

func (MDocNamespaceElementPresentValidator) ID() string {
	return "mdoc.namespace_element_present"
}

func (MDocNamespaceElementPresentValidator) Validate(_ context.Context, input Input) Result {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	presentation, ok := mdocPresentation(input.Value)
	if !ok {
		return wrongMDocInput(input.Value)
	}
	if document, exists := presentation.Document(pidMDocType); exists {
		if code, hasError := document.Errors[params.Namespace][params.Element]; hasError {
			return invalidMDocElement(
				params,
				fmt.Errorf("document contains ErrorItem code %d", code),
			)
		}
	}
	if _, exists := presentation.Element(params.Namespace, params.Element); !exists {
		return invalidMDocElement(params, fmt.Errorf("element is missing"))
	}
	return validMDocElement(params, "is present without an ErrorItem")
}

type JOSEJWEEncryptedResponseValidator struct{}

func (JOSEJWEEncryptedResponseValidator) ID() string {
	return "jose.jwe_encrypted_response"
}

func (JOSEJWEEncryptedResponseValidator) Validate(_ context.Context, input Input) Result {
	switch typed := input.Value.(type) {
	case string:
		if len(strings.Split(typed, ".")) == 5 {
			return Result{Status: StatusPass, Message: "compact JWE response is present"}
		}
		return Result{Status: StatusFail, Message: "response is not a compact JWE"}
	case map[string]any:
		if _, ok := typed["ciphertext"]; ok {
			return Result{Status: StatusPass, Message: "JSON JWE response is present"}
		}
		return Result{Status: StatusFail, Message: "response does not contain JWE ciphertext"}
	default:
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %T, expected string or object", input.Value),
		}
	}
}

type OID4VPDeviceBindingValidator struct{}

func (OID4VPDeviceBindingValidator) ID() string {
	return "oid4vp.device_binding"
}

func (OID4VPDeviceBindingValidator) Validate(_ context.Context, input Input) Result {
	if input.Value == nil {
		return Result{Status: StatusFail, Message: "device binding evidence is missing"}
	}
	switch typed := input.Value.(type) {
	case map[string]any:
		if hasAnyKey(
			typed,
			"key_binding",
			"keyBinding",
			"_kb_header",
			"device_signature",
			"deviceSignature",
		) {
			return Result{Status: StatusPass, Message: "device binding evidence is present"}
		}
		return Result{Status: StatusFail, Message: "device binding evidence is missing"}
	default:
		return Result{Status: StatusPass, Message: "device binding evidence is present"}
	}
}

type OID4VPNonceStateBindingValidator struct{}

func (OID4VPNonceStateBindingValidator) ID() string {
	return "oid4vp.nonce_state_binding"
}

func (OID4VPNonceStateBindingValidator) Validate(_ context.Context, input Input) Result {
	payload, ok := input.Value.(map[string]any)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %T, expected object", input.Value),
		}
	}
	requestNonce, _ := payload["request_nonce"].(string)
	responseNonce, _ := payload["response_nonce"].(string)
	requestState, _ := payload["request_state"].(string)
	responseState, _ := payload["response_state"].(string)

	if requestNonce != "" && responseNonce != "" && requestNonce != responseNonce {
		return Result{Status: StatusFail, Message: "nonce binding failed"}
	}
	if requestState != "" && responseState != "" && requestState != responseState {
		return Result{Status: StatusFail, Message: "state binding failed"}
	}
	return Result{Status: StatusPass, Message: "nonce and state bindings match"}
}

func hasAnyKey(object map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, ok := object[key]; ok {
			return true
		}
	}
	return false
}
