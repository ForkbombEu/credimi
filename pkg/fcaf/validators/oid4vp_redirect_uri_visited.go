// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"time"
)

const (
	redirectURIVisitedEventType       = "vp_redirect_uri_visited"
	presentationResponseEventType     = "vp_presentation_response_received"
	redirectURIVisitsRawEvidenceField = "redirect_uri_visits"
)

// OID4VPRedirectURIVisitedValidator proves that the Wallet's user agent opened
// the redirect URI the Verifier supplied after the Authorization Response. The
// capture service records a visit only when the request carries the
// response_code it generated for that session, so a recorded visit identifies
// the exact URI that was opened. The recorded envelope holds the request method
// and redacted headers only: it cannot show whether the Wallet appended further
// parameters to the URI.
type OID4VPRedirectURIVisitedValidator struct{}

func (OID4VPRedirectURIVisitedValidator) ID() string {
	return "oid4vp.redirect_uri_visited"
}

func (OID4VPRedirectURIVisitedValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		MinVisits                 int  `json:"min_visits"`
		AfterPresentationResponse bool `json:"after_presentation_response"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	minVisits := params.MinVisits
	if minVisits == 0 {
		minVisits = 1
	}
	if minVisits < 1 {
		return Result{Status: StatusError, Message: "min_visits must be positive"}
	}

	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "captured presentation session is not an object"}
	}

	count, ok := normalizeInteger(session["redirect_uri_visit_count"])
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: "capture recorded no redirect URI visit for this session",
		}
	}
	if count < minVisits {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"capture recorded %d redirect URI visit(s), expected at least %d",
				count,
				minVisits,
			),
		}
	}
	if normalizeString(session["redirect_uri_visited_at"]) == "" {
		return Result{
			Status:  StatusFail,
			Message: "capture recorded no redirect_uri_visited_at timestamp",
		}
	}

	visits, err := redirectURIVisitEnvelopes(session)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	if len(visits) < minVisits {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"capture retained %d redirect URI visit envelope(s), expected at least %d",
				len(visits),
				minVisits,
			),
		}
	}

	visitTimes, err := sessionEventTimes(session, redirectURIVisitedEventType)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	if len(visitTimes) < minVisits {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"capture recorded %d %q event(s), expected at least %d",
				len(visitTimes),
				redirectURIVisitedEventType,
				minVisits,
			),
		}
	}

	if params.AfterPresentationResponse {
		responseTimes, err := sessionEventTimes(session, presentationResponseEventType)
		if err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
		if len(responseTimes) == 0 {
			return Result{
				Status: StatusFail,
				Message: "capture recorded no Authorization Response, " +
					"so the redirect URI visit cannot follow one",
			}
		}
		if visitTimes[0].Before(responseTimes[0]) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"redirect URI was visited at %s, before the Authorization Response at %s",
					visitTimes[0].Format(time.RFC3339Nano),
					responseTimes[0].Format(time.RFC3339Nano),
				),
			}
		}
	}

	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"wallet opened the verifier-supplied redirect URI %d time(s)",
			count,
		),
	}
}

// redirectURIVisitEnvelopes returns the retained redirect-page request
// envelopes. Each envelope must carry the request method, which is what proves
// the record is a captured request rather than a placeholder.
func redirectURIVisitEnvelopes(session map[string]any) ([]map[string]any, error) {
	raw, ok := normalizeJSONObject(session["raw"])
	if !ok {
		return nil, fmt.Errorf("captured presentation session has no raw evidence")
	}
	entries, ok := raw[redirectURIVisitsRawEvidenceField].([]any)
	if !ok {
		return nil, fmt.Errorf(
			"captured presentation session has no raw.redirect_uri_visits record",
		)
	}
	envelopes := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		envelope, ok := normalizeJSONObject(entry)
		if !ok {
			return nil, fmt.Errorf("raw.redirect_uri_visits entry is not an object")
		}
		if normalizeString(envelope["method"]) == "" {
			return nil, fmt.Errorf("raw.redirect_uri_visits entry has no request method")
		}
		envelopes = append(envelopes, envelope)
	}
	return envelopes, nil
}

// sessionEventTimes returns the timestamps of one capture event type in
// recorded order.
func sessionEventTimes(session map[string]any, eventType string) ([]time.Time, error) {
	events, ok := session["events"].([]any)
	if !ok {
		return nil, fmt.Errorf("captured presentation session events are missing")
	}
	times := make([]time.Time, 0, len(events))
	for _, rawEvent := range events {
		event, ok := normalizeJSONObject(rawEvent)
		if !ok {
			continue
		}
		if normalizeString(event["type"]) != eventType {
			continue
		}
		at := normalizeString(event["at"])
		parsed, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return nil, fmt.Errorf(
				"capture event %q has an unparseable timestamp %q",
				eventType,
				at,
			)
		}
		times = append(times, parsed)
	}
	return times, nil
}
