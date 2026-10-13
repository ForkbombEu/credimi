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

const tempWalletVersionConfigKey = "temp_wallet_version"

func tempWalletVersionCleanupHook(
	ctx workflow.Context,
	_ *pipelineinternal.WorkflowDefinition,
	_ *workflow.ActivityOptions,
	config map[string]any,
	_ map[string]any,
	_ *map[string]any,
) error {
	cleanupConfig, ok := config[tempWalletVersionConfigKey].(map[string]any)
	if !ok {
		return nil
	}
	if !workflowengine.AsBool(cleanupConfig["cleanup"]) {
		return nil
	}

	recordID, _ := cleanupConfig["record_id"].(string)
	ownerID, _ := cleanupConfig["owner_id"].(string)
	identifier, _ := cleanupConfig["identifier"].(string)
	if recordID == "" {
		return nil
	}

	request := workflowengine.ActivityInput{
		Payload: activities.DeleteTempRecordInput{
			Collection:         "wallet_versions",
			RecordID:           recordID,
			ExpectedOwnerID:    ownerID,
			ExpectedIdentifier: identifier,
		},
	}

	cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
	return workflow.ExecuteActivity(cleanupCtx, activities.DeleteTempRecordActivityName, request).
		Get(cleanupCtx, nil)
}
