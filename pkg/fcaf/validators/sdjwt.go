// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"net/mail"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

type SDJWTClaimPresentValidator struct{}

func (SDJWTClaimPresentValidator) ID() string { return "sdjwt.claim_present" }

func (SDJWTClaimPresentValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	_, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", claim)}
	}
	return Result{Status: StatusPass, Message: fmt.Sprintf("claim %q is present", claim)}
}

type SDJWTClaimTypeValidator struct{}

func (SDJWTClaimTypeValidator) ID() string { return "sdjwt.claim_type" }

func (SDJWTClaimTypeValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim string `json:"claim"`
		Type  string `json:"type"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, params.Claim)
	if !ok {
		return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", params.Claim)}
	}
	if valueTypeName(value) != params.Type {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"claim %q is %s, expected %s",
				params.Claim,
				valueTypeName(value),
				params.Type,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("claim %q matches type %s", params.Claim, params.Type),
	}
}

type SDJWTClaimStringPrefixValidator struct{}

func (SDJWTClaimStringPrefixValidator) ID() string { return "sdjwt.claim_string_prefix" }

func (SDJWTClaimStringPrefixValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim  string `json:"claim"`
		Prefix string `json:"prefix"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Claim == "" {
		return Result{Status: StatusError, Message: "claim param is required"}
	}
	if params.Prefix == "" {
		return Result{Status: StatusError, Message: "prefix param is required"}
	}
	value, ok := sdjwtClaim(input.Value, params.Claim)
	if !ok {
		return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", params.Claim)}
	}
	text, ok := value.(string)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q is %T, expected string", params.Claim, value),
		}
	}
	if !strings.HasPrefix(text, params.Prefix) {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q does not start with %q", params.Claim, params.Prefix),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("claim %q starts with %q", params.Claim, params.Prefix),
	}
}

type SDJWTClaimUTF8StringValidator struct{}

func (SDJWTClaimUTF8StringValidator) ID() string { return "sdjwt.claim_utf8_string" }

func (SDJWTClaimUTF8StringValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", claim)}
	}
	text, ok := value.(string)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q is %T, expected string", claim, value),
		}
	}
	if !utf8.ValidString(text) {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q is not valid UTF-8", claim),
		}
	}
	if result := validateUTF8Vectors(input.Params); result != nil {
		return *result
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("claim %q is a valid UTF-8 string", claim),
	}
}

type SDJWTClaimRFC5322EmailValidator struct{}

func (SDJWTClaimRFC5322EmailValidator) ID() string { return "sdjwt.claim_rfc5322_email" }

func (SDJWTClaimRFC5322EmailValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", claim)}
	}
	text, ok := value.(string)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q is %T, expected string", claim, value),
		}
	}
	addr, err := mail.ParseAddress(text)
	if err != nil {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q is not a valid RFC 5322 address: %v", claim, err),
		}
	}
	if addr.Address != text || addr.Name != "" {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("claim %q must be a bare RFC 5322 addr-spec", claim),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("claim %q is a valid RFC 5322 address", claim),
	}
}

type PIDSDJWTVCTValidator struct{}

const pidSDJWTVCTPrefix = "urn:eudi:pid:"

func (PIDSDJWTVCTValidator) ID() string { return "pid.sdjwt_vct_pid" }

func (PIDSDJWTVCTValidator) Validate(_ context.Context, input Input) Result {
	value, ok := sdjwtClaim(input.Value, "vct")
	if !ok {
		return Result{Status: StatusFail, Message: `claim "vct" is missing`}
	}
	text, ok := value.(string)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf(`claim "vct" is %T, expected string`, value),
		}
	}
	if !strings.HasPrefix(text, pidSDJWTVCTPrefix) {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf(`claim "vct" does not start with %q`, pidSDJWTVCTPrefix),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf(`claim "vct" starts with %q`, pidSDJWTVCTPrefix),
	}
}

type PIDSDJWTMandatoryClaimsValidator struct{}

func (PIDSDJWTMandatoryClaimsValidator) ID() string {
	return "pid.sdjwt_required_mandatory_claims_present"
}

func (PIDSDJWTMandatoryClaimsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		RequiredElements []string `json:"required_elements"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(params.RequiredElements) == 0 {
		return Result{
			Status:  StatusError,
			Message: "required_elements param is required",
		}
	}

	missing := []string{}
	for _, claim := range params.RequiredElements {
		if _, ok := claimFromValue(input.Value, claim); !ok {
			missing = append(missing, claim)
		}
	}
	if len(missing) > 0 {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("required mandatory PID SD-JWT claims are missing: %v", missing),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: "all required mandatory PID SD-JWT claims are present",
	}
}

func decodeClaimParam(params map[string]any) (string, error) {
	decoded, err := DecodeParams[struct {
		Claim string `json:"claim"`
	}](params)
	if err != nil {
		return "", err
	}
	if decoded.Claim == "" {
		return "", fmt.Errorf("claim param is required")
	}
	return decoded.Claim, nil
}

func sdjwtClaim(value any, claim string) (any, bool) {
	var claims map[string]any
	switch typed := value.(type) {
	case *evidence.SDJWTPresentation:
		if typed == nil {
			return nil, false
		}
		claims = typed.Claims
	case evidence.SDJWTPresentation:
		claims = typed.Claims
	case map[string]any:
		claims = typed
	default:
		return nil, false
	}
	return resolveObjectPath(claims, claim)
}

func claimFromValue(value any, claim string) (any, bool) {
	if got, ok := sdjwtClaim(value, claim); ok {
		return got, true
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	if namespaces, ok := root["namespaces"].(map[string]any); ok {
		for _, rawNamespace := range namespaces {
			namespace, ok := rawNamespace.(map[string]any)
			if !ok {
				continue
			}
			if got, ok := namespace[claim]; ok {
				return got, true
			}
		}
	}
	return nil, false
}

func valueTypeName(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case float64, float32, int, int64, int32, uint, uint64, uint32:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		if value == nil {
			return "null"
		}
		return reflect.TypeOf(value).String()
	}
}

func validateUTF8Vectors(params map[string]any) *Result {
	vectors, err := decodeVectors(params)
	if err != nil {
		return &Result{Status: StatusError, Message: err.Error()}
	}
	if len(vectors.Positive) == 0 && len(vectors.Negative) == 0 {
		return nil
	}

	for _, path := range vectors.Positive {
		file, err := loadVectorFile(path)
		if err != nil {
			return &Result{Status: StatusError, Message: err.Error()}
		}
		for _, tc := range file.Cases {
			data, err := vectorCaseBytes(tc)
			if err != nil {
				return &Result{Status: StatusError, Message: err.Error()}
			}
			if !utf8.Valid(data) {
				return &Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("positive UTF-8 vector %q is invalid", tc.ID),
				}
			}
		}
	}

	for _, path := range vectors.Negative {
		file, err := loadVectorFile(path)
		if err != nil {
			return &Result{Status: StatusError, Message: err.Error()}
		}
		for _, tc := range file.Cases {
			data, err := vectorCaseBytes(tc)
			if err != nil {
				return &Result{Status: StatusError, Message: err.Error()}
			}
			if utf8.Valid(data) {
				return &Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("negative UTF-8 vector %q is valid", tc.ID),
				}
			}
		}
	}

	return nil
}
