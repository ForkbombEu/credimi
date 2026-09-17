// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"strings"
)

// OID4VPSessionEventCountValidator verifies how many times one capture event
// type was recorded on a presentation session. It is used to prove that the
// Wallet stopped communicating, for example that it delivered exactly one
// Authorization Response and then terminated the interaction.
type OID4VPSessionEventCountValidator struct{}

func (OID4VPSessionEventCountValidator) ID() string {
	return "oid4vp.session_event_count"
}

func (OID4VPSessionEventCountValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Type == "" {
		return Result{Status: StatusError, Message: "type param is required"}
	}
	if _, ok := input.Params["count"]; !ok {
		return Result{Status: StatusError, Message: "count param is required"}
	}

	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "captured presentation session is not an object"}
	}
	events, ok := session["events"].([]any)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: "captured presentation session events are missing",
		}
	}

	count := 0
	observed := make([]string, 0, len(events))
	for _, rawEvent := range events {
		event, ok := normalizeJSONObject(rawEvent)
		if !ok {
			continue
		}
		eventType, ok := event["type"].(string)
		if !ok {
			continue
		}
		observed = append(observed, eventType)
		if eventType == params.Type {
			count++
		}
	}

	if count != params.Count {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"capture recorded %d %q event(s), expected %d; observed events: %s",
				count,
				params.Type,
				params.Count,
				strings.Join(observed, ", "),
			),
		}
	}

	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"capture recorded exactly %d %q event(s)",
			params.Count,
			params.Type,
		),
	}
}
