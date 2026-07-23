// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
)

type CleanupMobileDeviceSemaphoreResourcesActivity struct {
	workflowengine.BaseActivity
}

type CleanupMobileDeviceSemaphoreResourcesActivityInput struct {
	AppURL  string                                                      `json:"app_url,omitempty"`
	Cleanup *mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata `json:"cleanup,omitempty"`
}

type CleanupMobileDeviceSemaphoreResourcesActivityOutput struct {
	CleanupFailures []string `json:"cleanup_failures,omitempty"`
}

func NewCleanupMobileDeviceSemaphoreResourcesActivity() *CleanupMobileDeviceSemaphoreResourcesActivity {
	return &CleanupMobileDeviceSemaphoreResourcesActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Cleanup mobile device semaphore resources",
		},
	}
}

func (a *CleanupMobileDeviceSemaphoreResourcesActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *CleanupMobileDeviceSemaphoreResourcesActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[CleanupMobileDeviceSemaphoreResourcesActivityInput](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	output := CleanupMobileDeviceSemaphoreResourcesActivityOutput{}
	if payload.Cleanup == nil {
		result.Output = output
		return result, nil
	}

	appURL := strings.TrimSpace(payload.AppURL)
	if appURL == "" {
		output.CleanupFailures = []string{"app_url missing for queued resource cleanup"}
		result.Output = output
		return result, nil
	}

	if recordID := strings.TrimSpace(payload.Cleanup.TempWalletVersionID); recordID != "" {
		output.CleanupFailures = append(output.CleanupFailures, a.deleteTempRecord(
			ctx,
			utils.JoinURL(appURL, "api", "wallet", "temp-version", recordID),
			map[string]any{
				"expected_owner_id":   payload.Cleanup.TempWalletVersionOwnerID,
				"expected_identifier": payload.Cleanup.TempWalletVersionIdentifier,
			},
			"temp wallet version",
			recordID,
		)...)
	}

	for _, credential := range payload.Cleanup.TempCredentials {
		recordID := strings.TrimSpace(credential.RecordID)
		if recordID == "" {
			continue
		}
		output.CleanupFailures = append(output.CleanupFailures, a.deleteTempRecord(
			ctx,
			utils.JoinURL(appURL, "api", "credential", "temp", recordID),
			map[string]any{
				"expected_owner_id":   credential.OwnerID,
				"expected_identifier": credential.Identifier,
			},
			"temp credential",
			recordID,
		)...)
	}

	for _, useCase := range payload.Cleanup.TempUseCaseVerifications {
		recordID := strings.TrimSpace(useCase.RecordID)
		if recordID == "" {
			continue
		}
		output.CleanupFailures = append(output.CleanupFailures, a.deleteTempRecord(
			ctx,
			utils.JoinURL(appURL, "api", "verifier", "temp-use-case", recordID),
			map[string]any{
				"expected_owner_id":   useCase.OwnerID,
				"expected_identifier": useCase.Identifier,
			},
			"temp use case verification",
			recordID,
		)...)
	}

	result.Output = output
	return result, nil
}

func (a *CleanupMobileDeviceSemaphoreResourcesActivity) deleteTempRecord(
	ctx context.Context,
	url string,
	body map[string]any,
	resourceKind string,
	recordID string,
) []string {
	result, err := executeInternalHTTPRequest(ctx, InternalHTTPActivityPayload{
		Method: http.MethodDelete,
		URL:    url,
		Body:   body,
	}, &a.BaseActivity)
	if err != nil {
		return []string{fmt.Sprintf("%s %s cleanup failed: %v", resourceKind, recordID, err)}
	}

	status, responseBody := decodeInternalHTTPStatus(result.Output)
	switch status {
	case http.StatusOK, http.StatusNotFound:
		return nil
	case 0:
		return []string{fmt.Sprintf("%s %s cleanup returned no status", resourceKind, recordID)}
	default:
		return []string{fmt.Sprintf(
			"%s %s cleanup failed with status %d: %v",
			resourceKind,
			recordID,
			status,
			responseBody,
		)}
	}
}

func decodeInternalHTTPStatus(output any) (int, any) {
	resultMap, ok := output.(map[string]any)
	if !ok {
		return 0, output
	}
	status := 0
	switch value := resultMap["status"].(type) {
	case int:
		status = value
	case int32:
		status = int(value)
	case int64:
		status = int(value)
	case float64:
		status = int(value)
	}
	return status, resultMap["body"]
}
