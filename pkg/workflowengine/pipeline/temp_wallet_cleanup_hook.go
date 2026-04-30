// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"net/http"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

const tempWalletVersionConfigKey = "temp_wallet_version"

func tempWalletVersionCleanupHook(
	ctx workflow.Context,
	_ []pipelineinternal.StepDefinition,
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
	appURL, _ := config["app_url"].(string)
	if recordID == "" || appURL == "" {
		return nil
	}

	internalHTTPActivity := activities.NewInternalHTTPActivity()
	request := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodDelete,
			URL: utils.JoinURL(
				appURL,
				"api", "wallet", "temp-version", recordID,
			),
			Body: map[string]any{
				"expected_owner_id":   ownerID,
				"expected_identifier": identifier,
			},
			ExpectedStatus: http.StatusOK,
		},
	}

	cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
	return workflow.ExecuteActivity(cleanupCtx, internalHTTPActivity.Name(), request).Get(cleanupCtx, nil)
}
