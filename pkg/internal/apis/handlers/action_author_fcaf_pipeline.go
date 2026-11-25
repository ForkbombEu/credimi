// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"

	"github.com/forkbombeu/credimi/pkg/fcaf/aggregate"
	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ActionAuthorFCAFPipelineRequest asks to build and run the FCAF
// relying-party validation of a wallet build from the wallet's own actions.
type ActionAuthorFCAFPipelineRequest struct {
	WalletVersionIdentifier string `json:"wallet_version_identifier" validate:"required"`
	DeviceID                string `json:"device_id"                 validate:"required"`
	// Scenarios limits the pipeline to these scenario file names of the
	// suite; empty runs every scenario.
	Scenarios []string `json:"scenarios"`
}

// ActionAuthorFCAFPipelineResponse is the built pipeline and its queued run.
type ActionAuthorFCAFPipelineResponse struct {
	// Pipeline is the pipeline record's path, organization/pipeline.
	Pipeline    string `json:"pipeline"`
	PipelineURL string `json:"pipeline_url"`
	TicketID    string `json:"ticket_id"`
	DeviceID    string `json:"device_id"`
	Status      string `json:"status"`
	WorkflowID  string `json:"workflow_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	RunURL      string `json:"run_url,omitempty"`
	// Skipped are the scenarios left out because the wallet has no written,
	// up-to-date action for some of their roles; never null.
	Skipped   []aggregate.SkippedScenario `json:"skipped"`
	TestCount int                         `json:"test_count"`
}

// HandleActionAuthorFCAFPipeline builds the FCAF relying-party validation
// pipeline of a wallet build from the caller organization's actions of the
// wallet, stores it on the wallet's FCAF pipeline record and queues it on the
// device, as POST /api/pipeline/queue does.
func HandleActionAuthorFCAFPipeline() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ActionAuthorFCAFPipelineRequest](e)
		if err != nil {
			return authoringAPIError(e, actionauthor.ErrInvalid)
		}
		if e.Auth == nil {
			return authoringAPIError(e, actionauthor.ErrForbidden)
		}
		if apiErr := requireAuthoringUser(e.Auth); apiErr != nil {
			return apiErr
		}
		actor := authoringActor(e.Auth)
		org, err := pbutils.GetUserOrganization(e.App, actor.ActorID)
		if err != nil {
			return apierror.New(
				http.StatusForbidden,
				"action_author",
				"forbidden",
				"you are not a member of any organization",
			)
		}
		deviceID := canonify.NormalizePath(input.DeviceID)
		if apiErr := validatePipelineRunnerAccess(
			e.App,
			org.Id,
			[]string{deviceID},
		); apiErr != nil {
			return apiErr
		}
		device, _, err := mobilerunner.ResolveDevice(e.App, deviceID)
		if err != nil {
			return actionAuthorDeviceNotFound(deviceID)
		}
		if actionAuthorPlatform(device.GetString("type")) != actionauthor.PlatformAndroid {
			return fcafNotSupported()
		}
		orgNamespace := org.GetString("canonified_name")
		selection, err := actionauthor.Select(e.App, orgNamespace, actionauthor.BuildRequest{
			Actor:                   actor,
			WalletVersionIdentifier: input.WalletVersionIdentifier,
			Platform:                actionauthor.PlatformAndroid,
		})
		if err != nil {
			return authoringAPIError(e, err)
		}
		scope, err := activities.LoadFCAFScope(actionauthor.FCAFRelyingPartySuite)
		if err != nil {
			return authoringAPIError(e, fmt.Errorf("load fcaf role contract: %w", err))
		}
		coverage, _, _, err := actionauthor.FCAFCoverage(e.App, selection, scope)
		if err != nil {
			return authoringAPIError(e, err)
		}
		paths, apiErr := fcafPipelineScenarios(e, scope, input.Scenarios)
		if apiErr != nil {
			return apiErr
		}
		subset := 0
		if len(input.Scenarios) > 0 {
			subset = len(paths)
		}
		walletName := selection.Wallet.GetString("name")
		versionID := selection.Identifier
		if selection.Source == actionauthor.SourceStore {
			// The store build is the wallet the device already holds.
			versionID = actionauthor.ExternalSourceVersion
		}
		built, err := aggregate.Build(scope.Root, paths, aggregate.Options{
			Suite: scope.Suite,
			Name:  actionauthor.FCAFPipelineName(walletName, subset),
			ActionPrefix: orgNamespace + "/" +
				selection.Wallet.GetString("canonified_name") + "/",
			VersionID: versionID,
			DeviceID:  deviceID,
			Roles:     actionauthor.FCAFWrittenRoles(coverage),
			// A subset keeps only the tests whose evidence it produces, as the
			// happy-flow pipeline does.
			FilterTestsByEvidence: subset > 0,
			NoVideo:               new(true),
		})
		if errors.Is(err, aggregate.ErrNothingToRun) {
			return apierror.New(
				http.StatusUnprocessableEntity,
				"action_author",
				"fcaf_nothing_to_run",
				"no FCAF scenario can run with the wallet's written actions ("+err.Error()+")",
			)
		}
		if err != nil {
			return authoringAPIError(e, fmt.Errorf("build FCAF pipeline: %w", err))
		}
		record, err := upsertFCAFPipeline(
			e.App,
			org.Id,
			actionauthor.FCAFPipelineName(walletName, subset),
			string(built.YAML),
		)
		if err != nil {
			return authoringAPIError(e, err)
		}
		identifier := orgNamespace + "/" + record.GetString("canonified_name")
		queued, apiErr := enqueuePipelineRun(e, pipelineQueueRunContext{
			pipelineRecord:     record,
			pipelineIdentifier: identifier,
			organizationRecord: org,
			userID:             e.Auth.Id,
			userName:           e.Auth.GetString("name"),
			userEmail:          e.Auth.GetString("email"),
			yaml:               string(built.YAML),
		})
		if apiErr != nil {
			return apiErr
		}
		skipped := built.Skipped
		if skipped == nil {
			skipped = []aggregate.SkippedScenario{}
		}
		return e.JSON(http.StatusOK, ActionAuthorFCAFPipelineResponse{
			Pipeline:    identifier,
			PipelineURL: queued.PipelineURL,
			TicketID:    queued.TicketID,
			DeviceID:    deviceID,
			Status:      string(queued.Status),
			WorkflowID:  queued.WorkflowID,
			RunID:       queued.RunID,
			RunURL:      queued.RunURL,
			Skipped:     skipped,
			TestCount:   len(built.TestIDs),
		})
	}
}

func fcafNotSupported() *apierror.APIError {
	return apierror.New(
		http.StatusUnprocessableEntity,
		"action_author",
		"fcaf_not_supported",
		"FCAF pipelines run on Android devices only",
	)
}

// fcafPipelineScenarios returns the suite's scenario files, limited to the
// named ones when names is not empty; an unknown name answers 422
// fcaf_unknown_scenario.
func fcafPipelineScenarios(
	e *core.RequestEvent,
	scope *actionauthor.FCAFScope,
	names []string,
) ([]string, *apierror.APIError) {
	paths, err := aggregate.ScenarioPaths(scope.Root, scope.Suite)
	if err != nil {
		return nil, authoringAPIError(e, err)
	}
	if len(names) == 0 {
		return paths, nil
	}
	selected := make([]string, 0, len(names))
	for _, path := range paths {
		if slices.Contains(names, filepath.Base(path)) {
			selected = append(selected, path)
		}
	}
	for _, name := range names {
		if !slices.ContainsFunc(selected, func(path string) bool {
			return filepath.Base(path) == name
		}) {
			return nil, apierror.New(
				http.StatusUnprocessableEntity,
				"action_author",
				"fcaf_unknown_scenario",
				"not a scenario of the FCAF "+scope.Suite+" suite: "+name,
			)
		}
	}
	return selected, nil
}

// upsertFCAFPipeline stores yaml on the organization's pipeline record
// named name, creating it unpublished when there is none. A published one
// cannot change and answers ErrConflict.
func upsertFCAFPipeline(app core.App, orgID, name, yaml string) (*core.Record, error) {
	record, err := app.FindFirstRecordByFilter(
		"pipelines",
		"owner = {:owner} && name = {:name}",
		dbx.Params{"owner": orgID, "name": name},
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		collection, err := app.FindCollectionByNameOrId("pipelines")
		if err != nil {
			return nil, fmt.Errorf("find pipelines collection: %w", err)
		}
		record = core.NewRecord(collection)
		record.Set("owner", orgID)
		record.Set("name", name)
		record.Set(
			"description",
			"The FCAF relying-party validation of the wallet, run with its own wallet actions.",
		)
		record.Set("published", false)
	case err != nil:
		return nil, fmt.Errorf("find FCAF pipeline: %w", err)
	case record.GetBool("published"):
		return nil, fmt.Errorf(
			"%w: the pipeline %q is published: unpublish it to run it again",
			actionauthor.ErrConflict,
			name,
		)
	}
	record.Set("yaml", yaml)
	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("save FCAF pipeline: %w", err)
	}
	return record, nil
}
