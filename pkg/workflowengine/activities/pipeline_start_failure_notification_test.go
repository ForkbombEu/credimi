// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/pipelinealerts"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"github.com/stretchr/testify/require"
)

func TestNotifyPipelineStartFailureActivityReachesOrganization(t *testing.T) {
	app := newPipelineResultsTestApp(t)
	_, identifier := createTestPipeline(t, app, "Wallet nightly")
	namespace, err := pbutils.GetOrganizationCanonifiedName(app, testOrgAID)
	require.NoError(t, err)

	member := subscriptions.NewDefaultClient()
	member.Subscribe(pipelinealerts.StartFailureTopic(testOrgAID))
	app.SubscriptionsBroker().Register(member)
	received := make(chan subscriptions.Message, 1)
	go func() { received <- <-member.Channel() }()

	_, err = executeActivity(
		t,
		NewNotifyPipelineStartFailureActivity(app),
		NotifyPipelineStartFailureInput{
			OwnerNamespace:     namespace,
			PipelineIdentifier: identifier,
			TicketID:           "ticket-1",
			ErrorMessage:       "search attribute ActionsID value size 2270 exceeds size limit 2048",
		},
	)
	require.NoError(t, err)

	select {
	case msg := <-received:
		require.Equal(t, pipelinealerts.StartFailureTopic(testOrgAID), msg.Name)
		var failure pipelinealerts.StartFailure
		require.NoError(t, json.Unmarshal(msg.Data, &failure))
		require.Equal(t, pipelinealerts.StartFailure{
			PipelineName:       "Wallet nightly",
			PipelineIdentifier: identifier,
			TicketID:           "ticket-1",
			Error:              "search attribute ActionsID value size 2270 exceeds size limit 2048",
		}, failure)
	case <-time.After(5 * time.Second):
		t.Fatal("organization subscriber did not receive the start failure")
	}
}

func TestNotifyPipelineStartFailureActivityUnknownOrganization(t *testing.T) {
	app := newPipelineResultsTestApp(t)

	_, err := executeActivity(
		t,
		NewNotifyPipelineStartFailureActivity(app),
		NotifyPipelineStartFailureInput{
			OwnerNamespace:     "no-such-organization",
			PipelineIdentifier: "no-such-organization/pipeline",
			TicketID:           "ticket-1",
		},
	)
	requireActivityError(t, err, errorcodes.RecordNotFound, true)
}
