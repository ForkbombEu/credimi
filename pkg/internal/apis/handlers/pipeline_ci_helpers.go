// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

const pipelineCIMobileAutomationStepUse = "mobile-automation"

var pipelineCIRunnerHealthCheck = checkPipelineCIRunnerHealth

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func metadataSHA(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	sha, _ := metadata["sha"].(string)
	return strings.TrimSpace(sha)
}

func ensureMetadataSHA(metadata map[string]any, sha string) map[string]any {
	if metadata == nil {
		metadata = map[string]any{}
	}
	if strings.TrimSpace(metadataSHA(metadata)) == "" && strings.TrimSpace(sha) != "" {
		metadata["sha"] = strings.TrimSpace(sha)
	}
	return metadata
}

func metadataFromForm(req *http.Request) (map[string]any, *apierror.APIError) {
	raw := strings.TrimSpace(req.FormValue("metadata"))
	if raw == "" {
		return nil, nil
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, apierror.New(
			http.StatusBadRequest,
			"metadata",
			"metadata must be valid JSON",
			err.Error(),
		)
	}
	return metadata, nil
}

func parsePipelineCIWorkflow(
	pipelineYAML string,
) (*pipelineinternal.WorkflowDefinition, *apierror.APIError) {
	workflowDefinition, err := pipelineinternal.ParseWorkflow(pipelineYAML)
	if err != nil {
		return nil, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse pipeline yaml",
			err.Error(),
		)
	}

	return workflowDefinition, nil
}

func injectPipelineCIGlobalRunnerID(
	pipelineYAML string,
	workflowDefinition *pipelineinternal.WorkflowDefinition,
	runnerID string,
	hasStepRunner bool,
	needsGlobalRunner bool,
) (string, *apierror.APIError) {
	runnerID = canonify.NormalizePath(runnerID)
	if runnerID == "" {
		return pipelineYAML, nil
	}

	if hasStepRunner {
		return "", apierror.New(
			http.StatusBadRequest,
			"runner_id",
			"runner_id cannot be combined with step runner_id",
			"remove step runner_id values or omit runner_id",
		)
	}
	if !needsGlobalRunner {
		return pipelineYAML, nil
	}

	workflowDefinition.Runtime.GlobalRunnerID = runnerID
	rewrittenYAML, err := yaml.Marshal(workflowDefinition)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"yaml",
			"failed to marshal pipeline yaml",
			err.Error(),
		)
	}

	return string(rewrittenYAML), nil
}

func pipelineCIMobileRunnerSelectionState(
	workflowDefinition *pipelineinternal.WorkflowDefinition,
) (bool, bool) {
	if workflowDefinition == nil {
		return false, false
	}

	hasStepRunner := false
	needsGlobalRunner := false
	check := func(step pipelineinternal.StepSpec) {
		if step.Use != pipelineCIMobileAutomationStepUse {
			return
		}
		runnerID, _ := step.With.Payload["runner_id"].(string)
		if strings.TrimSpace(runnerID) == "" {
			needsGlobalRunner = true
			return
		}
		hasStepRunner = true
	}

	for _, step := range workflowDefinition.Steps {
		check(step.StepSpec)
		for _, onErr := range step.OnError {
			if onErr != nil {
				check(onErr.StepSpec)
			}
		}
		for _, onSuccess := range step.OnSuccess {
			if onSuccess != nil {
				check(onSuccess.StepSpec)
			}
		}
	}

	return hasStepRunner, needsGlobalRunner
}

func selectPipelineCIRunnerByType(
	ctx context.Context,
	app core.App,
	runnerType string,
) (string, *apierror.APIError) {
	records, err := app.FindRecordsByFilter(
		"mobile_runners",
		"type = {:type} && published = true",
		"",
		-1,
		0,
		dbx.Params{"type": runnerType},
	)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"runner_type",
			"failed to list published runners",
			err.Error(),
		)
	}
	if len(records) == 0 {
		return "", apierror.New(
			http.StatusNotFound,
			"runner_type",
			"no published runner found for runner_type",
			"no published mobile runner matches "+runnerType,
		)
	}

	selectedRunnerID := ""
	selectedBacklog := 0
	for _, record := range records {
		online, apiErr := pipelineCIRunnerOnline(ctx, record)
		if apiErr != nil {
			return "", apiErr
		}
		if !online {
			continue
		}

		runnerID, apiErr := pipelineCIRunnerID(record, app)
		if apiErr != nil {
			return "", apiErr
		}
		backlog, apiErr := pipelineCIRunnerBacklog(ctx, runnerID)
		if apiErr != nil {
			return "", apiErr
		}
		if selectedRunnerID == "" ||
			backlog < selectedBacklog ||
			(backlog == selectedBacklog && runnerID < selectedRunnerID) {
			selectedRunnerID = runnerID
			selectedBacklog = backlog
		}
	}
	if selectedRunnerID == "" {
		return "", apierror.New(
			http.StatusServiceUnavailable,
			"runner_type",
			"no online published runner found for runner_type",
			"no online published mobile runner matches "+runnerType,
		)
	}

	return selectedRunnerID, nil
}

func pipelineCIRunnerID(record *core.Record, app core.App) (string, *apierror.APIError) {
	runnerID, err := mobileRunnerIdentifier(app, record)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"runner_id",
			"failed to build runner_id",
			err.Error(),
		)
	}
	if runnerID == "" {
		return "", apierror.New(
			http.StatusInternalServerError,
			"runner_id",
			"failed to build runner_id",
			"empty runner_id",
		)
	}

	return runnerID, nil
}

func pipelineCIRunnerOnline(ctx context.Context, record *core.Record) (bool, *apierror.APIError) {
	runnerURL := mobileRunnerURL(record)
	if runnerURL == "" {
		return false, nil
	}

	online, err := pipelineCIRunnerHealthCheck(ctx, runnerURL)
	if err != nil {
		return false, apierror.New(
			http.StatusInternalServerError,
			"runner_type",
			"failed to check runner health",
			err.Error(),
		)
	}

	return online, nil
}

func checkPipelineCIRunnerHealth(ctx context.Context, runnerURL string) (bool, error) {
	healthURL, err := url.JoinPath(runnerURL, "health")
	if err != nil {
		return false, err
	}

	healthCtx, cancel := context.WithTimeout(ctx, walletAPKRunnerHealthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(healthCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func pipelineCIRunnerBacklog(ctx context.Context, runnerID string) (int, *apierror.APIError) {
	state, err := queryMobileRunnerSemaphoreState(ctx, runnerID)
	if err != nil {
		if errors.Is(err, errSemaphoreNotFound) {
			return 0, nil
		}
		return 0, apierror.New(
			http.StatusInternalServerError,
			"runner_type",
			"failed to query runner queue",
			err.Error(),
		)
	}

	return state.QueueLen + state.SlotsUsed, nil
}
