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
const tempUseCaseVerificationsConfigKey = "temp_use_case_verifications"

func tempCredentialsCleanupHook(
	ctx workflow.Context,
	_ []pipelineinternal.StepDefinition,
	_ *workflow.ActivityOptions,
	config map[string]any,
	_ map[string]any,
	_ *map[string]any,
) error {
	return cleanupTempRecords(ctx, config, tempCredentialsConfigKey, "credentials", []string{
		"api",
		"credential",
		"temp",
	})
}

func tempUseCaseVerificationsCleanupHook(
	ctx workflow.Context,
	_ []pipelineinternal.StepDefinition,
	_ *workflow.ActivityOptions,
	config map[string]any,
	_ map[string]any,
	_ *map[string]any,
) error {
	return cleanupTempRecords(ctx, config, tempUseCaseVerificationsConfigKey, "use_cases", []string{
		"api",
		"verifier",
		"temp-use-case",
	})
}

func cleanupTempRecords(
	ctx workflow.Context,
	config map[string]any,
	configKey string,
	itemsKey string,
	urlParts []string,
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
	appURL, _ := config["app_url"].(string)
	if appURL == "" {
		return nil
	}

	internalHTTPActivity := activities.NewInternalHTTPActivity()
	cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
	for _, item := range items {
		recordID, _ := item["record_id"].(string)
		ownerID, _ := item["owner_id"].(string)
		identifier, _ := item["identifier"].(string)
		if recordID == "" {
			continue
		}
		recordURLParts := append(append([]string{}, urlParts...), recordID)
		request := workflowengine.ActivityInput{
			Payload: activities.InternalHTTPActivityPayload{
				Method: http.MethodDelete,
				URL:    utils.JoinURL(appURL, recordURLParts...),
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
