// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"github.com/stretchr/testify/require"
)

func TestSendRealtimeLogsActivity(t *testing.T) {
	app := newCredimiTestApp(t)
	const topic = "wf-1openid4vp-wallet-logs"

	subscriber := subscriptions.NewDefaultClient()
	subscriber.Subscribe(topic)
	app.SubscriptionsBroker().Register(subscriber)
	other := subscriptions.NewDefaultClient()
	other.Subscribe("another-topic")
	app.SubscriptionsBroker().Register(other)

	received := make(chan subscriptions.Message, 1)
	go func() { received <- <-subscriber.Channel() }()

	_, err := executeActivity(t, NewSendRealtimeLogsActivity(app), SendRealtimeLogsInput{
		Subscription: topic,
		Logs:         []map[string]any{{"message": "step-log"}},
	})
	require.NoError(t, err)

	select {
	case msg := <-received:
		require.Equal(t, topic, msg.Name)
		require.JSONEq(t, `[{"message":"step-log"}]`, string(msg.Data))
	case <-time.After(5 * time.Second):
		t.Fatal("subscriber did not receive the logs")
	}
	select {
	case msg := <-other.Channel():
		t.Fatalf("unexpected message on another topic: %s", msg.Name)
	default:
	}
}

func TestSendRealtimeLogsActivityRequiresSubscription(t *testing.T) {
	app := newCredimiTestApp(t)
	_, err := executeActivity(t, NewSendRealtimeLogsActivity(app), SendRealtimeLogsInput{})
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
}
