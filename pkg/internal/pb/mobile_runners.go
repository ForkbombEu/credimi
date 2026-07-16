// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
)

var mobileRunnerShutdownTemporalClient = temporalclient.GetTemporalClientWithNamespace

const mobileRunnerShutdownAcceptedTimeout = 5 * time.Second

func RegisterMobileRunnerHooks(app core.App) {
	bindMobileRunnerLifecycleMonitor(app)

	app.OnRecordAfterDeleteSuccess("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		runnerID, err := mobileRunnerRecordIdentifier(app, e.Record)
		if err != nil {
			return fmt.Errorf("build mobile runner identifier: %w", err)
		}

		temporalClient, err := mobileRunnerShutdownTemporalClient(
			workflowengine.MobileRunnerSemaphoreDefaultNamespace,
		)
		if err != nil {
			return fmt.Errorf("create semaphore temporal client: %w", err)
		}

		ctx, cancel := context.WithTimeout(
			context.Background(),
			mobileRunnerShutdownAcceptedTimeout,
		)
		defer cancel()

		_, err = temporalClient.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
			WorkflowID: workflows.MobileRunnerSemaphoreWorkflowID(runnerID),
			UpdateName: workflows.MobileRunnerSemaphoreShutdownRunnerUpdate,
			UpdateID:   "shutdown/" + runnerID,
			Args: []interface{}{
				workflows.MobileRunnerSemaphoreShutdownRunnerRequest{
					Reason: "mobile runner deleted",
				},
			},
			WaitForStage: tclient.WorkflowUpdateStageAccepted,
		})
		if err != nil {
			var notFound *serviceerror.NotFound
			if errors.As(err, &notFound) {
				return e.Next()
			}
			return fmt.Errorf("request semaphore workflow shutdown: %w", err)
		}

		return e.Next()
	})
}

func mobileRunnerRecordIdentifier(app core.App, record *core.Record) (string, error) {
	runnerID, err := canonify.BuildPath(
		app,
		record,
		canonify.CanonifyPaths["mobile_runners"],
		"",
	)
	if err != nil {
		return "", err
	}
	return canonify.NormalizePath(runnerID), nil
}
