// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package realtimelogs delivers workflow log updates to realtime subscribers.
package realtimelogs

import (
	"encoding/json"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
)

// Notify sends logs as JSON to every realtime client subscribed to
// subscription. Delivery is by topic name only; access control happens when
// clients subscribe.
func Notify(app core.App, subscription string, logs []map[string]any) error {
	rawData, err := json.Marshal(logs)
	if err != nil {
		return fmt.Errorf("marshal realtime logs: %w", err)
	}
	message := subscriptions.Message{
		Name: subscription,
		Data: rawData,
	}
	for _, client := range app.SubscriptionsBroker().Clients() {
		if client.HasSubscription(subscription) {
			client.Send(message)
		}
	}
	return nil
}
