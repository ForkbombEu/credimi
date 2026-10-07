// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

const tempCredentialsConfigKey = "temp_credentials"
const tempUseCaseVerificationsConfigKey = "temp_use_case_verifications"

func tempCredentialsCleanupHook(
	ctx workflow.Context,
	_ *pipelineinternal.WorkflowDefinition,
	_ *workflow.ActivityOptions,
	config map[string]any,
	_ map[string]any,
	_ *map[string]any,
) error {
	return cleanupTempRecords(ctx, config, tempCredentialsConfigKey, "credentials", "credentials")
}

func tempUseCaseVerificationsCleanupHook(
	ctx workflow.Context,
	_ *pipelineinternal.WorkflowDefinition,
	_ *workflow.ActivityOptions,
	config map[string]any,
	_ map[string]any,
	_ *map[string]any,
) error {
	return cleanupTempRecords(
		ctx,
		config,
		tempUseCaseVerificationsConfigKey,
		"use_cases",
		"use_cases_verifications",
	)
}

func cleanupTempRecords(
	ctx workflow.Context,
	config map[string]any,
	configKey string,
	itemsKey string,
	collection string,
) error {
	cleanupConfig, ok := config[configKey].(map[string]any)
	if !ok || !workflowengine.AsBool(cleanupConfig["cleanup"]) {
		return nil
	}

	rawItems, ok := cleanupConfig[itemsKey]
	if !ok {
		return nil
	}
	items := normalizeTempCredentialCleanupItems(rawItems)
	if len(items) == 0 {
		return nil
	}

	cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
	for _, item := range items {
		recordID, _ := item["record_id"].(string)
		ownerID, _ := item["owner_id"].(string)
		identifier, _ := item["identifier"].(string)
		if recordID == "" {
			continue
		}
		request := workflowengine.ActivityInput{
			Payload: activities.DeleteTempRecordInput{
				Collection:         collection,
				RecordID:           recordID,
				ExpectedOwnerID:    ownerID,
				ExpectedIdentifier: identifier,
			},
		}
		if err := workflow.ExecuteActivity(cleanupCtx, activities.DeleteTempRecordActivityName, request).
			Get(cleanupCtx, nil); err != nil {
			return err
		}
	}

	return nil
}

func normalizeTempCredentialCleanupItems(raw any) []map[string]any {
	switch values := raw.(type) {
	case []map[string]any:
		return values
	case []any:
		out := make([]map[string]any, 0, len(values))
		for _, value := range values {
			credential, ok := value.(map[string]any)
			if ok {
				out = append(out, credential)
			}
		}
		return out
	default:
		return nil
	}
}
