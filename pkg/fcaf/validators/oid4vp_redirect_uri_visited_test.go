// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// testRedirectVisitSession mirrors the capture session shape observed on the
// service: visit counters and timestamp at the top level, request envelopes
// under raw.redirect_uri_visits, and one event per visit.
func testRedirectVisitSession(
	responseAt string,
	visitTimes ...string,
) map[string]any {
	events := []any{
		map[string]any{"at": "2026-09-16T15:45:10.000Z", "type": "vp_session_created"},
	}
	if responseAt != "" {
		events = append(events, map[string]any{
			"at":   responseAt,
			"type": "vp_presentation_response_received",
		})
	}
	envelopes := make([]any, 0, len(visitTimes))
	for _, at := range visitTimes {
		events = append(events, map[string]any{"at": at, "type": "vp_redirect_uri_visited"})
		envelopes = append(envelopes, map[string]any{
			"method":  "GET",
			"headers": map[string]any{"user-agent": "wallet-browser"},
		})
	}
	session := map[string]any{
		"events": events,
		"raw":    map[string]any{"redirect_uri_visits": envelopes},
	}
	if len(visitTimes) > 0 {
		session["redirect_uri_visit_count"] = len(visitTimes)
		session["redirect_uri_visited_at"] = visitTimes[0]
	}
	return session
}

func TestOID4VPRedirectURIVisitedValidator(t *testing.T) {
	validator := OID4VPRedirectURIVisitedValidator{}

	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{
			name:   "visit recorded after the authorization response",
			value:  testRedirectVisitSession("2026-09-16T15:45:12.000Z", "2026-09-16T15:45:15.750Z"),
			params: map[string]any{"after_presentation_response": true},
			status: StatusPass,
		},
		{
			name:   "wallet never opened the redirect uri",
			value:  testRedirectVisitSession("2026-09-16T15:45:12.000Z"),
			params: map[string]any{},
			status: StatusFail,
		},
		{
			name:   "visit predates the authorization response",
			value:  testRedirectVisitSession("2026-09-16T15:45:20.000Z", "2026-09-16T15:45:15.750Z"),
			params: map[string]any{"after_presentation_response": true},
			status: StatusFail,
		},
		{
			name:   "no authorization response was captured",
			value:  testRedirectVisitSession("", "2026-09-16T15:45:15.750Z"),
			params: map[string]any{"after_presentation_response": true},
			status: StatusFail,
		},
		{
			name:   "ordering is not checked unless requested",
			value:  testRedirectVisitSession("2026-09-16T15:45:20.000Z", "2026-09-16T15:45:15.750Z"),
			params: map[string]any{},
			status: StatusPass,
		},
		{
			name: "repeated visits satisfy a higher minimum",
			value: testRedirectVisitSession(
				"2026-09-16T15:45:12.000Z",
				"2026-09-16T15:45:15.750Z",
				"2026-09-16T15:45:19.100Z",
			),
			params: map[string]any{"min_visits": 2},
			status: StatusPass,
		},
		{
			name:   "single visit fails a higher minimum",
			value:  testRedirectVisitSession("2026-09-16T15:45:12.000Z", "2026-09-16T15:45:15.750Z"),
			params: map[string]any{"min_visits": 2},
			status: StatusFail,
		},
		{
			name: "counter without a retained envelope is not evidence",
			value: map[string]any{
				"redirect_uri_visit_count": 1,
				"redirect_uri_visited_at":  "2026-09-16T15:45:15.750Z",
				"events": []any{
					map[string]any{"at": "2026-09-16T15:45:15.750Z", "type": "vp_redirect_uri_visited"},
				},
				"raw": map[string]any{},
			},
			params: map[string]any{},
			status: StatusFail,
		},
		{
			name: "counter without a capture event is not evidence",
			value: map[string]any{
				"redirect_uri_visit_count": 1,
				"redirect_uri_visited_at":  "2026-09-16T15:45:15.750Z",
				"events":                   []any{},
				"raw": map[string]any{
					"redirect_uri_visits": []any{map[string]any{"method": "GET", "headers": map[string]any{}}},
				},
			},
			params: map[string]any{},
			status: StatusFail,
		},
		{
			name:   "min_visits must be positive",
			value:  testRedirectVisitSession("2026-09-16T15:45:12.000Z", "2026-09-16T15:45:15.750Z"),
			params: map[string]any{"min_visits": -1},
			status: StatusError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  tt.value,
				Params: tt.params,
			})
			require.Equal(t, tt.status, result.Status, result.Message)
		})
	}
}
