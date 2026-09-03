// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/githubapp"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	workflowpipeline "github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
)

var signalGitHubPRCommentUpdate = activities.SignalGitHubPRCommentUpdate

func buildWalletAPKGitHubPRNotification(
	metadata map[string]any,
	appURL string,
	pipelineIdentifier string,
	deviceID string,
	deviceType string,
) *workflows.MobileDeviceSemaphoreNotification {
	return buildPipelineGitHubPRNotification(
		metadata,
		appURL,
		pipelineIdentifier,
		deviceID,
		deviceType,
		activities.GitHubPRCommentSectionWalletAPK,
	)
}

func buildPipelineGitHubPRNotification(
	metadata map[string]any,
	appURL string,
	pipelineIdentifier string,
	deviceID string,
	deviceType string,
	sectionTitle string,
) *workflows.MobileDeviceSemaphoreNotification {
	repository := metadataString(metadata, "repository")
	prNumber := pullRequestNumberFromMetadata(metadata)
	if repository == "" || prNumber <= 0 {
		return nil
	}

	return &workflows.MobileDeviceSemaphoreNotification{
		GitHubPR: &workflows.MobileDeviceSemaphoreGitHubPRNotification{
			Repository:        repository,
			PullRequestNumber: prNumber,
			CommitSHA: firstNonEmpty(
				metadataString(metadata, "event.pull_request.head.sha"),
				metadataSHA(metadata),
			),
			PipelineIdentifier: pipelineIdentifier,
			DeviceID:           deviceID,
			DeviceType:         deviceType,
			DeviceTypes:        buildInitialGitHubPRDeviceTypes(deviceID, deviceType),
			PipelineURL:        buildPipelinePageURL(appURL, pipelineIdentifier),
			AppURL:             appURL,
			SectionTitle:       sectionTitle,
		},
	}
}

func buildInitialGitHubPRDeviceTypes(deviceID string, deviceType string) map[string]string {
	deviceID = strings.TrimSpace(deviceID)
	deviceType = strings.TrimSpace(deviceType)
	if deviceID == "" || deviceType == "" {
		return nil
	}
	return map[string]string{deviceID: deviceType}
}

func maybeCreateWalletAPKQueuedPRComment(
	ctx context.Context,
	notification *workflows.MobileDeviceSemaphoreNotification,
	response PipelineRunWalletAPKResponse,
) error {
	return maybeCreatePipelineGitHubPRComment(ctx, notification, response.PipelineQueueResponse)
}

func maybeCreatePipelineGitHubPRComment(
	ctx context.Context,
	notification *workflows.MobileDeviceSemaphoreNotification,
	response PipelineQueueResponse,
) error {
	if notification == nil || notification.GitHubPR == nil {
		return nil
	}
	return signalGitHubPRCommentUpdate(ctx, activities.UpdateGitHubPRCommentInput{
		Repository:        notification.GitHubPR.Repository,
		PullRequestNumber: notification.GitHubPR.PullRequestNumber,
		CommitSHA:         notification.GitHubPR.CommitSHA,
		Status:            string(response.Status),
		Position:          response.Position,
		PipelineID:        notification.GitHubPR.PipelineIdentifier,
		DeviceID: githubPRCommentDeviceID(
			notification.GitHubPR.DeviceID,
			response.DeviceIDs,
		),
		DeviceType:   githubPRCommentDeviceType(notification.GitHubPR, response.DeviceIDs),
		PipelineURL:  notification.GitHubPR.PipelineURL,
		AppURL:       notification.GitHubPR.AppURL,
		WorkflowID:   response.WorkflowID,
		RunID:        response.RunID,
		TicketID:     response.TicketID,
		ErrorMessage: response.ErrorMessage,
		SectionTitle: notification.GitHubPR.SectionTitle,
	})
}

func buildPipelineGitHubPRCommentConfig(
	notification *workflows.MobileDeviceSemaphoreNotification,
) map[string]any {
	if notification == nil || notification.GitHubPR == nil {
		return nil
	}
	return map[string]any{
		workflowpipeline.GitHubPRCommentConfigRepositoryKey:        notification.GitHubPR.Repository,
		workflowpipeline.GitHubPRCommentConfigPullRequestNumberKey: notification.GitHubPR.PullRequestNumber,
		workflowpipeline.GitHubPRCommentConfigCommitSHAKey:         notification.GitHubPR.CommitSHA,
		workflowpipeline.GitHubPRCommentConfigPipelineIDKey:        notification.GitHubPR.PipelineIdentifier,
		workflowpipeline.GitHubPRCommentConfigPipelineURLKey:       notification.GitHubPR.PipelineURL,
		workflowpipeline.GitHubPRCommentConfigAppURLKey:            notification.GitHubPR.AppURL,
		workflowpipeline.GitHubPRCommentConfigSectionTitleKey:      notification.GitHubPR.SectionTitle,
	}
}

func githubPRCommentDeviceType(
	notification *workflows.MobileDeviceSemaphoreGitHubPRNotification,
	deviceIDs []string,
) string {
	if notification == nil {
		return ""
	}
	deviceID := githubPRCommentDeviceID(notification.DeviceID, deviceIDs)
	if deviceType := strings.TrimSpace(notification.DeviceTypes[deviceID]); deviceType != "" {
		return deviceType
	}
	return notification.DeviceType
}

func buildGitHubPRDeviceTypes(
	app core.App,
	deviceIDs []string,
	existing map[string]string,
) map[string]string {
	deviceTypes := map[string]string{}
	for deviceID, deviceType := range existing {
		if strings.TrimSpace(deviceID) != "" && strings.TrimSpace(deviceType) != "" {
			deviceTypes[deviceID] = deviceType
		}
	}
	for _, deviceID := range deviceIDs {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" || strings.TrimSpace(deviceTypes[deviceID]) != "" {
			continue
		}
		if deviceType := resolveWalletAPKGitHubPRDeviceType(app, deviceID, ""); deviceType != "" {
			deviceTypes[deviceID] = deviceType
		}
	}
	if len(deviceTypes) == 0 {
		return nil
	}
	return deviceTypes
}

func githubPRCommentDeviceID(deviceID string, deviceIDs []string) string {
	if strings.TrimSpace(deviceID) != "" {
		return deviceID
	}
	if len(deviceIDs) == 0 {
		return ""
	}
	return deviceIDs[0]
}

func pullRequestNumberFromMetadata(metadata map[string]any) int {
	return githubapp.IntFromAny(metadataValue(metadata, "event.number"))
}

func metadataString(metadata map[string]any, key string) string {
	value, _ := metadataValue(metadata, key).(string)
	return strings.TrimSpace(value)
}

func metadataValue(metadata map[string]any, key string) any {
	if metadata == nil {
		return nil
	}
	if value, ok := metadata[key]; ok {
		return value
	}
	parts := strings.Split(key, ".")
	var current any = metadata
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}
