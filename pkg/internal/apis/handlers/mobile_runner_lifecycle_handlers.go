// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
)

const (
	defaultMobileRunnerHeartbeatTimeoutSeconds = 120
	defaultMobileRunnerShutdownAfterSeconds    = 7 * 24 * 60 * 60
)

var mobileRunnerLifecycleNow = func() time.Time {
	return time.Now().UTC()
}

var mobileRunnerLifecycleTemporalClient = temporalclient.GetTemporalClientWithNamespace

type MobileRunnerLifecycleRequest struct {
	RunnerID string `json:"runner_id" validate:"required"`
	Reason   string `json:"reason,omitempty"`
}

type MobileRunnerLifecycleResponse struct {
	RunnerID                string `json:"runner_id"`
	Online                  bool   `json:"online"`
	SemaphoreWorkflowID     string `json:"semaphore_workflow_id"`
	HeartbeatTimeoutSeconds int    `json:"heartbeat_timeout_seconds"`
	ShutdownAfterSeconds    int    `json:"shutdown_after_seconds"`
}

var MobileRunnerLifecycleRoutes = routing.RouteGroup{
	BaseURL:                "/api/mobile-runner/lifecycle",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodPost,
			Path:           "/resume",
			Handler:        HandleMobileRunnerLifecycleResume,
			RequestSchema:  MobileRunnerLifecycleRequest{},
			ResponseSchema: MobileRunnerLifecycleResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/heartbeat",
			Handler:        HandleMobileRunnerLifecycleHeartbeat,
			RequestSchema:  MobileRunnerLifecycleRequest{},
			ResponseSchema: MobileRunnerLifecycleResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/pause",
			Handler:        HandleMobileRunnerLifecyclePause,
			RequestSchema:  MobileRunnerLifecycleRequest{},
			ResponseSchema: MobileRunnerLifecycleResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
	},
}

func HandleMobileRunnerLifecycleResume() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[MobileRunnerLifecycleRequest](e)
		if err != nil {
			return apierror.New(http.StatusBadRequest, "mobile_runner", "invalid_request", err.Error())
		}

		record, runnerID, apiErr := resolveLifecycleRunner(e.App, e.Auth, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}

		now := mobileRunnerLifecycleNow()
		setRunnerHeartbeat(record, true, now)
		if err := e.App.Save(record); err != nil {
			return apierror.New(http.StatusInternalServerError, "mobile_runner", "failed_to_save_mobile_runner", err.Error())
		}

		if err := ensureRunQueueSemaphoreWorkflowTemporal(e.Request.Context(), runnerID); err != nil {
			return apierror.New(http.StatusInternalServerError, "mobile_runner", "failed_to_ensure_runner_semaphore", err.Error())
		}

		_, err = updateRunnerSemaphore(
			e.Request.Context(),
			runnerID,
			workflows.MobileRunnerSemaphoreResumeRunnerUpdate,
			workflows.MobileRunnerSemaphoreResumeRunnerRequest{Reason: lifecycleReason(input.Reason, "runner_startup")},
			nil,
			lifecycleUpdateID("resume", runnerID),
		)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "mobile_runner", "failed_to_resume_runner_semaphore", err.Error())
		}

		return e.JSON(http.StatusOK, lifecycleResponse(runnerID, true))
	}
}

func HandleMobileRunnerLifecycleHeartbeat() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[MobileRunnerLifecycleRequest](e)
		if err != nil {
			return apierror.New(http.StatusBadRequest, "mobile_runner", "invalid_request", err.Error())
		}

		record, runnerID, apiErr := resolveLifecycleRunner(e.App, e.Auth, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}

		setRunnerHeartbeat(record, true, mobileRunnerLifecycleNow())
		if err := e.App.Save(record); err != nil {
			return apierror.New(http.StatusInternalServerError, "mobile_runner", "failed_to_save_mobile_runner", err.Error())
		}

		return e.JSON(http.StatusOK, lifecycleResponse(runnerID, true))
	}
}

func HandleMobileRunnerLifecyclePause() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[MobileRunnerLifecycleRequest](e)
		if err != nil {
			return apierror.New(http.StatusBadRequest, "mobile_runner", "invalid_request", err.Error())
		}

		record, runnerID, apiErr := resolveLifecycleRunner(e.App, e.Auth, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}

		record.Set("online", false)
		if err := e.App.Save(record); err != nil {
			return apierror.New(http.StatusInternalServerError, "mobile_runner", "failed_to_save_mobile_runner", err.Error())
		}

		_, err = updateRunnerSemaphore(
			e.Request.Context(),
			runnerID,
			workflows.MobileRunnerSemaphorePauseRunnerUpdate,
			workflows.MobileRunnerSemaphorePauseRunnerRequest{
				Reason:               lifecycleReason(input.Reason, "runner_shutdown"),
				CancelRunning:        true,
				ShutdownAfterSeconds: defaultMobileRunnerShutdownAfterSeconds,
			},
			nil,
			lifecycleUpdateID("pause", runnerID),
		)
		if err != nil && !errors.Is(err, errSemaphoreNotFound) {
			return apierror.New(http.StatusInternalServerError, "mobile_runner", "failed_to_pause_runner_semaphore", err.Error())
		}

		return e.JSON(http.StatusOK, lifecycleResponse(runnerID, false))
	}
}

func resolveLifecycleRunner(
	app core.App,
	auth *core.Record,
	runnerID string,
) (*core.Record, string, *apierror.APIError) {
	normalizedRunnerID := canonify.NormalizePath(runnerID)
	if normalizedRunnerID == "" {
		return nil, "", apierror.New(http.StatusBadRequest, "runner_id", "runner_id_required", "runner_id is required")
	}

	record, err := canonify.Resolve(app, normalizedRunnerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", apierror.New(http.StatusNotFound, "runner_id", "mobile_runner_not_found", "mobile runner not found")
		}
		return nil, "", apierror.New(http.StatusInternalServerError, "runner_id", "failed_to_resolve_runner_id", err.Error())
	}
	if record.Collection() == nil || record.Collection().Name != "mobile_runners" {
		return nil, "", apierror.New(http.StatusBadRequest, "runner_id", "invalid_runner_id", "runner_id does not reference a mobile runner")
	}

	if !isSuperuserAuth(auth) {
		orgID, err := pbutils.GetUserOrganizationID(app, auth.Id)
		if err != nil {
			return nil, "", apierror.New(http.StatusInternalServerError, "organization", "failed_to_find_user_organization", err.Error())
		}
		if record.GetString("owner") != orgID {
			return nil, "", apierror.New(http.StatusForbidden, "runner_id", "runner_owner_mismatch", "runner_id does not belong to the authenticated organization")
		}
	}

	canonicalRunnerID, err := mobileRunnerIdentifier(app, record)
	if err != nil {
		return nil, "", apierror.New(http.StatusInternalServerError, "runner_id", "failed_to_build_runner_id", err.Error())
	}

	return record, canonicalRunnerID, nil
}

func updateRunnerSemaphore(
	ctx context.Context,
	runnerID string,
	updateName string,
	req any,
	out any,
	updateID string,
) (bool, error) {
	client, err := mobileRunnerLifecycleTemporalClient(workflowengine.MobileRunnerSemaphoreDefaultNamespace)
	if err != nil {
		return false, err
	}

	handle, err := client.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID:   workflows.MobileRunnerSemaphoreWorkflowID(runnerID),
		UpdateName:   updateName,
		UpdateID:     updateID,
		Args:         []any{req},
		WaitForStage: tclient.WorkflowUpdateStageAccepted,
	})
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return false, errSemaphoreNotFound
		}
		return false, err
	}
	if out != nil {
		if err := handle.Get(ctx, out); err != nil {
			return true, err
		}
		return true, nil
	}
	return true, nil
}

func setRunnerHeartbeat(record *core.Record, online bool, now time.Time) {
	record.Set("online", online)
	record.Set("last_heartbeat_at", now.UTC().Format("2006-01-02 15:04:05.000Z"))
}

func lifecycleResponse(
	runnerID string,
	online bool,
) MobileRunnerLifecycleResponse {
	return MobileRunnerLifecycleResponse{
		RunnerID:                runnerID,
		Online:                  online,
		SemaphoreWorkflowID:     workflows.MobileRunnerSemaphoreWorkflowID(runnerID),
		HeartbeatTimeoutSeconds: defaultMobileRunnerHeartbeatTimeoutSeconds,
		ShutdownAfterSeconds:    defaultMobileRunnerShutdownAfterSeconds,
	}
}

func lifecycleReason(reason string, fallback string) string {
	reason = strings.TrimSpace(reason)
	if reason != "" {
		return reason
	}
	return fallback
}

func lifecycleUpdateID(action string, runnerID string) string {
	return fmt.Sprintf("%s/%s/%d", action, runnerID, mobileRunnerLifecycleNow().UnixNano())
}
