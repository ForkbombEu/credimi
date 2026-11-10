// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package pipelinealerts delivers pipeline start failures to the realtime
// clients of an organization. Nothing is stored: a client that is not
// connected when the failure happens does not receive it.
package pipelinealerts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
)

// StartFailureTopicPrefix starts the realtime topic of an organization's
// pipeline start failures: `pipeline_start_failures/<organization id>`.
const StartFailureTopicPrefix = "pipeline_start_failures/"

// StartFailure is the realtime message of a pipeline run that could not start.
type StartFailure struct {
	PipelineName       string `json:"pipeline_name"`
	PipelineIdentifier string `json:"pipeline_identifier"`
	TicketID           string `json:"ticket_id"`
	Error              string `json:"error"`
}

// StartFailureTopic returns the realtime topic of orgID's start failures.
func StartFailureTopic(orgID string) string {
	return StartFailureTopicPrefix + orgID
}

// StartFailureTopicOrgID returns the organization ID of a start failure topic,
// and false when topic is not one.
func StartFailureTopicOrgID(topic string) (string, bool) {
	orgID, ok := strings.CutPrefix(topic, StartFailureTopicPrefix)
	if !ok {
		return "", false
	}
	return orgID, true
}

// NotifyStartFailure sends failure to every realtime client subscribed to the
// start failure topic of orgID. Delivery is by topic name only: the realtime
// subscribe hook decides who may hold the topic.
func NotifyStartFailure(app core.App, orgID string, failure StartFailure) error {
	data, err := json.Marshal(failure)
	if err != nil {
		return fmt.Errorf("marshal pipeline start failure: %w", err)
	}
	topic := StartFailureTopic(orgID)
	message := subscriptions.Message{Name: topic, Data: data}
	for _, client := range app.SubscriptionsBroker().Clients() {
		if client.HasSubscription(topic) {
			client.Send(message)
		}
	}
	return nil
}
