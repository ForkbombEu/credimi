// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

// realtimeLogsTestWorkflows lists the workflows each organization namespace owns.
var realtimeLogsTestWorkflows = map[string][]string{
	"usera-s-organization": {"wf-a", "wf-ewc-status"},
	"userb-s-organization": {"wf-b"},
}

func stubRealtimeLogsTemporalClient(t *testing.T) {
	t.Helper()

	orig := complianceTemporalClient
	t.Cleanup(func() { complianceTemporalClient = orig })

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		owned := realtimeLogsTestWorkflows[namespace]
		mockClient := &temporalmocks.Client{}
		mockClient.
			On("DescribeWorkflowExecution", mock.Anything, mock.Anything, "").
			Return(func(
				_ context.Context,
				workflowID string,
				_ string,
			) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				if slices.Contains(owned, workflowID) {
					return &workflowservice.DescribeWorkflowExecutionResponse{}, nil
				}
				return nil, serviceerror.NewNotFound("workflow not found")
			}).
			Maybe()
		return mockClient, nil
	}
}

func realtimeAuthToken(t testing.TB, collection string, email string) string {
	t.Helper()

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	record, err := app.FindAuthRecordByEmail(collection, email)
	require.NoError(t, err)
	token, err := record.NewAuthToken()
	require.NoError(t, err)
	return token
}

func TestRealtimeLogsAuthorizationHook(t *testing.T) {
	stubRealtimeLogsTemporalClient(t)

	userAToken := realtimeAuthToken(t, "users", "userA@example.org")
	userBToken := realtimeAuthToken(t, "users", "userB@example.org")
	superuserToken := realtimeAuthToken(t, core.CollectionNameSuperusers, "admin@example.org")

	userATopics := []string{
		"wf-aopenidnet-logs",
		"wf-a-logs",
		"wf-ewcewc-logs",
		"wf-aeudiw-logs",
	}

	scenarios := []struct {
		name     string
		token    string
		topics   []string
		expected []string
	}{
		{
			name:     "guest holds no log topics and keeps collection topics",
			topics:   append([]string{"users", "pipeline_results/abc"}, userATopics...),
			expected: []string{"users", "pipeline_results/abc"},
		},
		{
			name:     "user of another organization holds no log topics",
			token:    userBToken,
			topics:   append([]string{"wf-bopenidnet-logs"}, userATopics...),
			expected: []string{"wf-bopenidnet-logs"},
		},
		{
			name:     "user of the owning organization keeps its log topics",
			token:    userAToken,
			topics:   userATopics,
			expected: userATopics,
		},
		{
			name:     "log topics of unowned workflows and bare suffixes are stripped",
			token:    userAToken,
			topics:   []string{"wf-bopenidnet-logs", "openidnet-logs", "-logs"},
			expected: []string{},
		},
		{
			name:     "superuser keeps log topics",
			token:    superuserToken,
			topics:   userATopics,
			expected: userATopics,
		},
	}

	for _, s := range scenarios {
		realtimeClient := subscriptions.NewDefaultClient()

		headers := map[string]string{}
		if s.token != "" {
			headers["Authorization"] = s.token
		}

		quotedTopics := make([]string, 0, len(s.topics))
		for _, topic := range s.topics {
			quotedTopics = append(quotedTopics, fmt.Sprintf("%q", topic))
		}

		scenario := tests.ApiScenario{
			Name:   s.name,
			Method: http.MethodPost,
			URL:    "/api/realtime",
			Body: strings.NewReader(fmt.Sprintf(
				`{"clientId":%q,"subscriptions":[%s]}`,
				realtimeClient.Id(),
				strings.Join(quotedTopics, ","),
			)),
			Headers:        headers,
			ExpectedStatus: http.StatusNoContent,
			ExpectedEvents: map[string]int{"*": 0, "OnRealtimeSubscribeRequest": 1},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app, err := tests.NewTestApp(testDataDir)
				require.NoError(t, err)
				RegisterRealtimeLogsAuthorizationHook(app)
				app.SubscriptionsBroker().Register(realtimeClient)
				return app
			},
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
				require.ElementsMatch(
					t,
					s.expected,
					slices.Collect(maps.Keys(realtimeClient.Subscriptions())),
				)
			},
		}
		scenario.Test(t)
	}
}

func TestNotifyLogsUpdateReachesOnlyAuthorizedSubscribers(t *testing.T) {
	stubRealtimeLogsTemporalClient(t)

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	userA, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	userB, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)

	const topic = "wf-aopenidnet-logs"
	subscribe := func(auth *core.Record) *subscriptions.DefaultClient {
		c := subscriptions.NewDefaultClient()
		c.Subscribe(authorizedRealtimeSubscriptions(t.Context(), app, auth, []string{topic})...)
		app.SubscriptionsBroker().Register(c)
		return c
	}
	owner := subscribe(userA)
	guest := subscribe(nil)
	otherOrg := subscribe(userB)

	received := make(chan subscriptions.Message, 1)
	go func() { received <- <-owner.Channel() }()

	// Send blocks on unbuffered client channels, so a delivery to the guest
	// or to the other organization would hang notifyLogsUpdate.
	done := make(chan error, 1)
	go func() {
		done <- notifyLogsUpdate(app, topic, []map[string]any{{"message": "secret-step-log"}})
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("notifyLogsUpdate delivered to an unauthorized subscriber")
	}

	msg := <-received
	require.Equal(t, topic, msg.Name)
	require.JSONEq(t, `[{"message":"secret-step-log"}]`, string(msg.Data))
	require.False(t, guest.HasSubscription(topic))
	require.False(t, otherOrg.HasSubscription(topic))
}
