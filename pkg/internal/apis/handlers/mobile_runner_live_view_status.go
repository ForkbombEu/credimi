// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
)

// MobileRunnerLiveViewStatusResponse tells a runner whether the workflow that
// holds a live-viewed device still runs.
type MobileRunnerLiveViewStatusResponse struct {
	Running bool `json:"running"`
}

// HandleMobileRunnerLifecycleLiveView answers whether a live view a runner
// serves still belongs to a running workflow that uses this runner: an action
// author session in the runner's own namespace that still holds one of the
// runner's devices (the session's release clears them), or a pipeline run of
// an organization whose pipeline_results row lists one of the runner's
// devices. Anything else, a missing workflow included, is not running. Only the
// runner's managers ask (authorizeRunnerManager): owners of its organization,
// and superusers or the internal admin key for an admin-managed runner.
func HandleMobileRunnerLifecycleLiveView() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		query := e.Request.URL.Query()
		runner, _, apiErr := authorizeRunnerManager(
			e.App,
			e.Auth,
			query.Get("runner_id"),
			"only organization owners can check a runner's live view",
			"only the internal admin key can check a live view of an admin-managed runner",
		)
		if apiErr != nil {
			return apiErr
		}
		namespace := query.Get("namespace")
		workflowID := query.Get("workflow_id")
		runID := query.Get("run_id")
		if namespace == "" || workflowID == "" {
			return e.JSON(http.StatusOK, MobileRunnerLiveViewStatusResponse{})
		}

		var wantType string
		switch {
		case namespace == mobilerunner.RunnerNamespace(runner.Id):
			wantType = activities.ActionAuthorSessionWorkflowName
		default:
			// Only organization namespaces get a client: the runner chooses
			// the namespace.
			if _, err := pbutils.FindOrganizationByNamespace(e.App, namespace); err != nil {
				return e.JSON(http.StatusOK, MobileRunnerLiveViewStatusResponse{})
			}
			wantType = pipeline.NewPipelineWorkflow().Name()
		}

		running, err := workflowRunning(
			e.Request.Context(),
			namespace,
			workflowID,
			runID,
			wantType,
		)
		if err != nil {
			return temporalUnavailableError()
		}
		if !running {
			return e.JSON(http.StatusOK, MobileRunnerLiveViewStatusResponse{})
		}
		if wantType == activities.ActionAuthorSessionWorkflowName {
			c, err := mobileRunnerLifecycleTemporalClient(namespace)
			if err != nil {
				return temporalUnavailableError()
			}
			running, err = actionAuthorSessionHoldsDevice(
				e.Request.Context(),
				e.App,
				c,
				workflowID,
				runID,
				runner.Id,
			)
			if err != nil {
				return temporalUnavailableError()
			}
		} else {
			running, err = pipelineRunUsesRunner(e.App, workflowID, runID, runner.Id)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"mobile_runner",
					"pipeline_result_lookup_failed",
					err.Error(),
				)
			}
		}
		return e.JSON(http.StatusOK, MobileRunnerLiveViewStatusResponse{Running: running})
	}
}

// workflowRunning reports whether the run runID of workflowID in namespace is
// running and of type wantType. A missing workflow is not running.
func workflowRunning(
	ctx context.Context,
	namespace, workflowID, runID, wantType string,
) (bool, error) {
	c, err := mobileRunnerLifecycleTemporalClient(namespace)
	if err != nil {
		return false, err
	}
	description, err := temporalclient.Describe(ctx, c, workflowID, runID)
	if err != nil {
		var notFound *serviceerror.NotFound
		var namespaceNotFound *serviceerror.NamespaceNotFound
		if errors.As(err, &notFound) || errors.As(err, &namespaceNotFound) {
			return false, nil
		}
		return false, err
	}
	info := description.GetWorkflowExecutionInfo()
	return info.GetType().GetName() == wantType &&
		info.GetStatus() == enums.WORKFLOW_EXECUTION_STATUS_RUNNING, nil
}

// pipelineRunUsesRunner reports whether the pipeline_results row of a pipeline
// run lists a device hosted by the runner with record ID runnerID.
func pipelineRunUsesRunner(app core.App, workflowID, runID, runnerID string) (bool, error) {
	result, err := pipelineresults.FindByWorkflowRun(app, workflowID, runID)
	if err != nil {
		if errors.Is(err, pipelineresults.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	deviceIDs := result.GetStringSlice("devices")
	if len(deviceIDs) == 0 {
		return false, nil
	}
	devices, err := app.FindRecordsByIds("mobile_devices", deviceIDs)
	if err != nil {
		return false, err
	}
	for _, device := range devices {
		if device.GetString("runner") == runnerID {
			return true, nil
		}
	}
	return false, nil
}
