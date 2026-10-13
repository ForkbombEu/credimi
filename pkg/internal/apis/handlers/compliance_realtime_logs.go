// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

// realtimeLogTopicSuffix is the suffix shared by every workflow log topic.
const realtimeLogTopicSuffix = "-logs"

// realtimeLogTopics lists each log topic suffix with the suffixes its producers
// trim from the workflow ID before building the topic, so the owning workflow
// is `<topic without suffix><workflowIDSuffix>`. The first matching suffix
// wins, so the generic suffix must stay last.
var realtimeLogTopics = []struct {
	suffix             string
	workflowIDSuffixes []string
}{
	// sendOpenID4VPWalletLogUpdateStart and the OpenID4VP wallet child trim "-log".
	{suffix: workflows.OpenID4VPWalletSubscription, workflowIDSuffixes: []string{"", "-log"}},
	// ewcLikeLogsSubscription and the EWC status workflow trim "-status".
	{suffix: workflows.EWCSubscription, workflowIDSuffixes: []string{"", "-status"}},
	{suffix: workflows.EudiwSubscription, workflowIDSuffixes: []string{""}},
	// HandleMyWorkflowLogs channel.
	{suffix: realtimeLogTopicSuffix, workflowIDSuffixes: []string{""}},
}

// RegisterRealtimeLogsAuthorizationHook strips realtime subscriptions to
// workflow log topics that the requester's organization does not own.
// realtimelogs.Notify delivers by topic name only, so this hook is the access
// decision for log streams: guests hold no log topics, and users only hold
// topics of workflows that exist in their organization namespace.
func RegisterRealtimeLogsAuthorizationHook(app core.App) {
	app.OnRealtimeSubscribeRequest().BindFunc(func(e *core.RealtimeSubscribeRequestEvent) error {
		e.Subscriptions = authorizedRealtimeSubscriptions(
			e.Request.Context(),
			e.App,
			e.Auth,
			e.Subscriptions,
		)
		return e.Next()
	})
}

func authorizedRealtimeSubscriptions(
	ctx context.Context,
	app core.App,
	auth *core.Record,
	topics []string,
) []string {
	allowed := make([]string, 0, len(topics))
	var temporalClient client.Client
	clientResolved := false

	for _, topic := range topics {
		workflowIDs := realtimeLogTopicWorkflowIDs(topic)
		if workflowIDs == nil {
			allowed = append(allowed, topic)
			continue
		}
		if auth == nil {
			continue
		}
		if auth.IsSuperuser() {
			allowed = append(allowed, topic)
			continue
		}
		if !clientResolved {
			clientResolved = true
			temporalClient = realtimeLogsTemporalClient(app, auth)
		}
		if temporalClient != nil && anyWorkflowExists(ctx, app, temporalClient, workflowIDs) {
			allowed = append(allowed, topic)
		}
	}

	return allowed
}

// realtimeLogTopicWorkflowIDs returns the workflow IDs a log topic may belong
// to, or nil when the topic is not a workflow log topic. PocketBase collection
// and record topics never end with "-logs": collection names and record IDs
// cannot contain "-".
func realtimeLogTopicWorkflowIDs(topic string) []string {
	if !strings.HasSuffix(topic, realtimeLogTopicSuffix) {
		return nil
	}

	workflowIDs := []string{}
	for _, logTopic := range realtimeLogTopics {
		base, ok := strings.CutSuffix(topic, logTopic.suffix)
		if !ok {
			continue
		}
		if base == "" {
			return workflowIDs
		}
		for _, workflowIDSuffix := range logTopic.workflowIDSuffixes {
			workflowIDs = append(workflowIDs, base+workflowIDSuffix)
		}
		return workflowIDs
	}
	return workflowIDs
}

func realtimeLogsTemporalClient(app core.App, auth *core.Record) client.Client {
	namespace, err := pbutils.GetUserOrganizationCanonifiedName(app, auth.Id)
	if err != nil || namespace == "" {
		return nil
	}
	temporalClient, err := complianceTemporalClient(namespace)
	if err != nil {
		app.Logger().Warn(
			"realtime logs authorization: unable to create temporal client",
			"namespace", namespace,
			"error", err,
		)
		return nil
	}
	return temporalClient
}

func anyWorkflowExists(
	ctx context.Context,
	app core.App,
	temporalClient client.Client,
	workflowIDs []string,
) bool {
	for _, workflowID := range workflowIDs {
		_, err := temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
		if err == nil {
			return true
		}
		notFound := &serviceerror.NotFound{}
		if !errors.As(err, &notFound) {
			app.Logger().Warn(
				"realtime logs authorization: unable to describe workflow",
				"workflow_id", workflowID,
				"error", err,
			)
		}
	}
	return false
}
