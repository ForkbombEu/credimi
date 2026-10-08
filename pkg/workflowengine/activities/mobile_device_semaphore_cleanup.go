// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/pocketbase/pocketbase/core"
)

type CleanupMobileDeviceSemaphoreResourcesActivity struct {
	workflowengine.BaseActivity
	app core.App
}

type CleanupMobileDeviceSemaphoreResourcesActivityInput struct {
	Cleanup *mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata `json:"cleanup,omitempty"`
}

type CleanupMobileDeviceSemaphoreResourcesActivityOutput struct {
	CleanupFailures []string `json:"cleanup_failures,omitempty"`
}

// CleanupMobileDeviceSemaphoreResourcesActivityName is the registered name of
// CleanupMobileDeviceSemaphoreResourcesActivity.
const CleanupMobileDeviceSemaphoreResourcesActivityName = "Cleanup mobile device semaphore resources"

func NewCleanupMobileDeviceSemaphoreResourcesActivity(
	app core.App,
) *CleanupMobileDeviceSemaphoreResourcesActivity {
	return &CleanupMobileDeviceSemaphoreResourcesActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: CleanupMobileDeviceSemaphoreResourcesActivityName,
		},
		app: app,
	}
}

func (a *CleanupMobileDeviceSemaphoreResourcesActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *CleanupMobileDeviceSemaphoreResourcesActivity) Execute(
	_ context.Context,
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

	if recordID := strings.TrimSpace(payload.Cleanup.TempWalletVersionID); recordID != "" {
		output.CleanupFailures = a.appendCleanupFailure(
			output.CleanupFailures,
			"wallet_versions",
			recordID,
			payload.Cleanup.TempWalletVersionOwnerID,
			payload.Cleanup.TempWalletVersionIdentifier,
		)
	}

	for _, credential := range payload.Cleanup.TempCredentials {
		recordID := strings.TrimSpace(credential.RecordID)
		if recordID == "" {
			continue
		}
		output.CleanupFailures = a.appendCleanupFailure(
			output.CleanupFailures,
			"credentials",
			recordID,
			credential.OwnerID,
			credential.Identifier,
		)
	}

	for _, useCase := range payload.Cleanup.TempUseCaseVerifications {
		recordID := strings.TrimSpace(useCase.RecordID)
		if recordID == "" {
			continue
		}
		output.CleanupFailures = a.appendCleanupFailure(
			output.CleanupFailures,
			"use_cases_verifications",
			recordID,
			useCase.OwnerID,
			useCase.Identifier,
		)
	}

	result.Output = output
	return result, nil
}

func (a *CleanupMobileDeviceSemaphoreResourcesActivity) appendCleanupFailure(
	failures []string,
	collection, recordID, ownerID, identifier string,
) []string {
	if _, err := deleteTempRecord(a.app, collection, recordID, ownerID, identifier); err != nil {
		return append(failures, fmt.Sprintf("delete %s %s: %v", collection, recordID, err))
	}
	return failures
}
