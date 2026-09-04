// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

// SDJWTCompactSerializationValidator verifies RFC 9901 tilde-delimited
// presentation framing with or without a final Key Binding JWT.
type SDJWTCompactSerializationValidator struct{}

func (SDJWTCompactSerializationValidator) ID() string {
	return "sdjwt.compact_serialization"
}

func (SDJWTCompactSerializationValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		KeyBinding bool `json:"key_binding"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if _, found := input.Params["key_binding"]; !found {
		return Result{Status: StatusError, Message: "key_binding is required"}
	}

	presentations, ok := sdjwtPresentations(input.Value)
	if !ok || len(presentations) == 0 {
		return Result{Status: StatusFail, Message: "SD-JWT presentation evidence is missing"}
	}
	for index, presentation := range presentations {
		if result := validateSDJWTCompactSerialization(
			presentation,
			params.KeyBinding,
		); result != nil {
			result.Message = fmt.Sprintf("presentation[%d]: %s", index, result.Message)
			return *result
		}
	}

	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"all %d SD-JWT presentations use compact serialization",
			len(presentations),
		),
		Details: map[string]any{
			"key_binding":        params.KeyBinding,
			"presentation_count": len(presentations),
		},
	}
}

func validateSDJWTCompactSerialization(
	presentation *evidence.SDJWTPresentation,
	keyBinding bool,
) *Result {
	if presentation == nil || presentation.Raw == "" || presentation.SDJWT == "" {
		return &Result{Status: StatusFail, Message: "compact SD-JWT bytes are missing"}
	}
	issuerJWT, _, found := strings.Cut(presentation.SDJWT, "~")
	if !found || strings.Count(issuerJWT, ".") != 2 {
		return &Result{
			Status:  StatusFail,
			Message: "compact serialization must start with an issuer-signed JWT and a tilde",
		}
	}
	if !strings.HasSuffix(presentation.SDJWT, "~") {
		return &Result{
			Status:  StatusFail,
			Message: "SD-JWT and disclosure sequence must end with a tilde",
		}
	}

	if keyBinding {
		if presentation.KeyBindingJWT == "" {
			return &Result{Status: StatusFail, Message: "compact presentation is missing a KB-JWT"}
		}
		if strings.Count(presentation.KeyBindingJWT, ".") != 2 {
			return &Result{Status: StatusFail, Message: "KB-JWT is not a compact JWT"}
		}
		if presentation.Raw != presentation.SDJWT+presentation.KeyBindingJWT {
			return &Result{
				Status:  StatusFail,
				Message: "KB-JWT is not the final compact serialization component",
			}
		}
		return nil
	}

	if presentation.KeyBindingJWT != "" {
		return &Result{
			Status:  StatusFail,
			Message: "compact presentation unexpectedly contains a KB-JWT",
		}
	}
	if presentation.Raw != presentation.SDJWT {
		return &Result{
			Status:  StatusFail,
			Message: "compact presentation bytes do not match SD-JWT bytes",
		}
	}
	return nil
}
