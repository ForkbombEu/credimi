// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/stretchr/testify/require"
)

// statusListInput builds the presented SD-JWT VC evidence shape with one member
// of the status_list status mechanism set to value.
func statusListInput(member string, value any) Input {
	return Input{
		Value: &evidence.SDJWTPresentation{Claims: map[string]any{
			"status": map[string]any{"status_list": map[string]any{member: value}},
		}},
		Params: map[string]any{"claim": "status.status_list." + member},
	}
}

func TestSDJWTClaimNonNegativeIntegerValidator(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		status Status
	}{
		{name: "zero float64", value: float64(0), status: StatusPass},
		{name: "positive float64", value: float64(3), status: StatusPass},
		{name: "positive int", value: 3, status: StatusPass},
		{name: "negative float64", value: float64(-1), status: StatusFail},
		{name: "non integral float64", value: 1.5, status: StatusFail},
		{name: "numeric string", value: "3", status: StatusFail},
		{name: "boolean", value: true, status: StatusFail},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SDJWTClaimNonNegativeIntegerValidator{}.Validate(
				context.Background(),
				statusListInput("idx", test.value),
			)

			require.Equal(t, test.status, result.Status)
		})
	}
}

func TestSDJWTClaimNonNegativeIntegerValidatorMissingClaim(t *testing.T) {
	result := SDJWTClaimNonNegativeIntegerValidator{}.Validate(
		context.Background(),
		statusListInput("uri", "https://issuer.example/status-list/1"),
	)

	require.Equal(t, StatusFail, result.Status)
}

func TestSDJWTClaimNonNegativeIntegerValidatorRequiresClaimParam(t *testing.T) {
	input := statusListInput("idx", float64(1))
	input.Params = map[string]any{"claim": ""}

	result := SDJWTClaimNonNegativeIntegerValidator{}.Validate(context.Background(), input)

	require.Equal(t, StatusError, result.Status)
}

func TestSDJWTClaimURIValidator(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		status Status
	}{
		{
			name:   "https URI",
			value:  "https://issuer.example/status-list/1",
			status: StatusPass,
		},
		{name: "urn URI", value: "urn:example:status-list:1", status: StatusPass},
		{name: "relative reference", value: "/status/1", status: StatusFail},
		{name: "empty string", value: "", status: StatusFail},
		{name: "non string", value: 42, status: StatusFail},
		{name: "unparseable string", value: "http://[::1", status: StatusFail},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SDJWTClaimURIValidator{}.Validate(
				context.Background(),
				statusListInput("uri", test.value),
			)

			require.Equal(t, test.status, result.Status)
		})
	}
}

func TestSDJWTClaimURIValidatorMissingClaim(t *testing.T) {
	result := SDJWTClaimURIValidator{}.Validate(
		context.Background(),
		statusListInput("idx", float64(1)),
	)

	require.Equal(t, StatusFail, result.Status)
}

func TestSDJWTClaimURIValidatorRequiresClaimParam(t *testing.T) {
	input := statusListInput("uri", "https://issuer.example/status-list/1")
	input.Params = map[string]any{"claim": ""}

	result := SDJWTClaimURIValidator{}.Validate(context.Background(), input)

	require.Equal(t, StatusError, result.Status)
}
