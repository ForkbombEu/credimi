// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package webpush

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	webpushgo "github.com/SherClockHolmes/webpush-go"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	orgAuthorizationsCollection = "orgAuthorizations"
	pushSubscriptionsCollection = "push_subscriptions"

	// pushTTLSeconds keeps a notification deliverable for one hour so
	// devices that are briefly offline still receive it.
	pushTTLSeconds = 3600

	// pushSendTimeout bounds a single push send so a stalled push service
	// cannot hang the whole notification fan-out.
	pushSendTimeout = 10 * time.Second
)

// CompletionRequest describes a finished pipeline run to notify users about.
type CompletionRequest struct {
	OrgID        string
	PipelineName string
	Organization string
	WorkflowID   string
	RunID        string
	Result       string
	Duration     string
	Error        string
	AppURL       string
}

type pushPayload struct {
	PipelineName string `json:"pipeline_name"`
	Organization string `json:"organization,omitempty"`
	Result       string `json:"result"`
	Duration     string `json:"duration,omitempty"`
	Error        string `json:"error,omitempty"`
	URL          string `json:"url"`
}

type subscriptionKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// NotifyPipelineRunCompletion sends a Web Push notification about a finished
// pipeline run to every push subscription of the organization members.
// Per-subscription failures are collected into the returned error without
// aborting the remaining sends. Dead subscriptions (HTTP 404/410) are pruned.
func NotifyPipelineRunCompletion(
	ctx context.Context,
	app core.App,
	req CompletionRequest,
) (int, error) {
	publicKey, privateKey, err := GetVAPIDKeyPair(app)
	if err != nil {
		return 0, err
	}

	subscriptions, err := findOrgMemberSubscriptions(app, req.OrgID)
	if err != nil {
		return 0, err
	}

	payload, err := json.Marshal(pushPayload{
		PipelineName: req.PipelineName,
		Organization: req.Organization,
		Result:       req.Result,
		Duration:     req.Duration,
		Error:        req.Error,
		URL:          buildPipelineRunURL(req.AppURL, req.WorkflowID, req.RunID),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal push payload: %w", err)
	}

	sent := 0
	var sendErrs []error
	for _, subscription := range subscriptions {
		sendCtx, cancel := context.WithTimeout(ctx, pushSendTimeout)
		delivered, err := sendPush(
			sendCtx,
			app,
			subscription,
			payload,
			publicKey,
			privateKey,
			req.AppURL,
		)
		cancel()
		if err != nil {
			sendErrs = append(sendErrs, err)
			continue
		}
		if delivered {
			sent++
		}
	}
	return sent, errors.Join(sendErrs...)
}

func buildPipelineRunURL(appURL string, workflowID string, runID string) string {
	return utils.JoinURL(appURL, "my", "tests", "runs", workflowID, runID)
}

func findOrgMemberSubscriptions(app core.App, orgID string) ([]*core.Record, error) {
	members, err := app.FindRecordsByFilter(
		orgAuthorizationsCollection,
		"organization = {:org}",
		"",
		0,
		0,
		dbx.Params{"org": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve organization members: %w", err)
	}

	params := dbx.Params{"org": orgID}
	conditions := make([]string, 0, len(members))
	for i, member := range members {
		userID := member.GetString("user")
		if userID == "" {
			continue
		}
		key := fmt.Sprintf("user%d", i)
		params[key] = userID
		conditions = append(conditions, fmt.Sprintf("user = {:%s}", key))
	}
	if len(conditions) == 0 {
		return nil, nil
	}

	subscriptions, err := app.FindRecordsByFilter(
		pushSubscriptionsCollection,
		strings.Join(conditions, " || "),
		"",
		0,
		0,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch push subscriptions: %w", err)
	}
	return subscriptions, nil
}

func sendPush(
	ctx context.Context,
	app core.App,
	record *core.Record,
	payload []byte,
	publicKey string,
	privateKey string,
	appURL string,
) (bool, error) {
	var keys subscriptionKeys
	if err := record.UnmarshalJSONField("keys", &keys); err != nil {
		return false, fmt.Errorf("invalid push subscription keys: %w", err)
	}

	subscription := webpushgo.Subscription{
		Endpoint: record.GetString("endpoint"),
		Keys: webpushgo.Keys{
			P256dh: keys.P256dh,
			Auth:   keys.Auth,
		},
	}
	resp, err := webpushgo.SendNotificationWithContext(
		ctx,
		payload,
		&subscription,
		&webpushgo.Options{
			Subscriber:      appURL,
			TTL:             pushTTLSeconds,
			VAPIDPublicKey:  publicKey,
			VAPIDPrivateKey: privateKey,
		},
	)
	if err != nil {
		return false, fmt.Errorf("failed to send push notification: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return true, nil
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		if err := app.Delete(record); err != nil {
			return false, fmt.Errorf("failed to prune dead push subscription: %w", err)
		}
		return false, nil
	default:
		return false, fmt.Errorf("push service returned status: %s", resp.Status)
	}
}
