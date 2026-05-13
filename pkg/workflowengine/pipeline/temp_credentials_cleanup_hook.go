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

const tempCredentialsConfigKey = "temp_credentials"

func tempCredentialsCleanupHook(
	ctx workflow.Context,
	_ []pipelineinternal.StepDefinition,
	_ *workflow.ActivityOptions,
	config map[string]any,
	_ map[string]any,
	_ *map[string]any,
) error {
	cleanupConfig, ok := config[tempCredentialsConfigKey].(map[string]any)
	if !ok || !workflowengine.AsBool(cleanupConfig["cleanup"]) {
		return nil
	}

	rawCredentials, ok := cleanupConfig["credentials"]
	if !ok {
		return nil
	}
	credentials := normalizeTempCredentialCleanupItems(rawCredentials)
	if len(credentials) == 0 {
		return nil
	}
	appURL, _ := config["app_url"].(string)
	if appURL == "" {
		return nil
	}

	internalHTTPActivity := activities.NewInternalHTTPActivity()
	cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
	for _, credential := range credentials {
		recordID, _ := credential["record_id"].(string)
		ownerID, _ := credential["owner_id"].(string)
		identifier, _ := credential["identifier"].(string)
		if recordID == "" {
			continue
		}
		request := workflowengine.ActivityInput{
			Payload: activities.InternalHTTPActivityPayload{
				Method: http.MethodDelete,
				URL: utils.JoinURL(
					appURL,
					"api", "credential", "temp", recordID,
				),
				Body: map[string]any{
					"expected_owner_id":   ownerID,
					"expected_identifier": identifier,
				},
				ExpectedStatus: http.StatusOK,
			},
		}
		if err := workflow.ExecuteActivity(cleanupCtx, internalHTTPActivity.Name(), request).
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
