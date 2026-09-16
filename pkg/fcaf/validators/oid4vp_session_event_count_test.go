// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPSessionEventCountValidator(t *testing.T) {
	validator := OID4VPSessionEventCountValidator{}
	events := func(types ...string) map[string]any {
		list := make([]any, 0, len(types))
		for _, eventType := range types {
			list = append(list, map[string]any{"type": eventType, "detail": map[string]any{}})
		}
		return map[string]any{"events": list}
	}

	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{
			name:   "single response delivery",
			value:  events("vp_session_created", "vp_deeplink_generated", "vp_presentation_response_received"),
			params: map[string]any{"type": "vp_presentation_response_received", "count": 1},
			status: StatusPass,
		},
		{
			name:   "repeated response delivery",
			value:  events("vp_session_created", "vp_presentation_response_received", "vp_presentation_response_received"),
			params: map[string]any{"type": "vp_presentation_response_received", "count": 1},
			status: StatusFail,
		},
		{
			name:   "wallet never delivered a response",
			value:  events("vp_session_created", "vp_deeplink_generated"),
			params: map[string]any{"type": "vp_presentation_response_received", "count": 1},
			status: StatusFail,
		},
		{
			name:   "zero is an explicit expectation",
			value:  events("vp_session_created"),
			params: map[string]any{"type": "vp_presentation_response_received", "count": 0},
			status: StatusPass,
		},
		{
			name:   "events are missing",
			value:  map[string]any{},
			params: map[string]any{"type": "vp_presentation_response_received", "count": 1},
			status: StatusFail,
		},
		{
			name:   "type param missing",
			value:  events("vp_session_created"),
			params: map[string]any{"count": 1},
			status: StatusError,
		},
		{
			name:   "count param missing",
			value:  events("vp_session_created"),
			params: map[string]any{"type": "vp_session_created"},
			status: StatusError,
		},
		{
			name:   "evidence is not an object",
			value:  []any{},
			params: map[string]any{"type": "vp_session_created", "count": 1},
			status: StatusFail,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.Validate(context.Background(), Input{Value: tt.value, Params: tt.params})
			require.Equal(t, tt.status, got.Status, got.Message)
		})
	}
}
