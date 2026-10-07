// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelineresults

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const collectionName = "pipeline_results"

var (
	ErrNotFound     = errors.New("pipeline result not found")
	ErrConflict     = errors.New("pipeline result already exists for another pipeline")
	ErrInvalidInput = errors.New("invalid pipeline result input")
)

// CreateInput identifies a pipeline run whose result record must exist.
type CreateInput struct {
	OwnerID    string
	PipelineID string
	WorkflowID string
	RunID      string
	RunType    string
	DeviceIDs  []string
}

// FindByWorkflowRun returns the pipeline result of a workflow run.
func FindByWorkflowRun(app core.App, workflowID, runID string) (*core.Record, error) {
	record, err := app.FindFirstRecordByFilter(
		collectionName,
		"workflow_id = {:workflow_id} && run_id = {:run_id}",
		dbx.Params{
			"workflow_id": workflowID,
			"run_id":      runID,
		},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(
				"%w: workflow_id %s run_id %s",
				ErrNotFound,
				workflowID,
				runID,
			)
		}
		return nil, fmt.Errorf("lookup pipeline result: %w", err)
	}
	return record, nil
}

// Create stores the pipeline result of a run. It is idempotent on
// (workflow_id, run_id): an existing record for the same owner and pipeline is
// returned, one for another owner or pipeline returns ErrConflict.
func Create(app core.App, in CreateInput) (*core.Record, error) {
	runType := strings.TrimSpace(in.RunType)
	if runType == "" {
		runType = pipelineinternal.RunTypeManual
	}
	if !pipelineinternal.ValidRunType(runType) {
		return nil, fmt.Errorf(
			"%w: type must be one of %q, %q, or %q",
			ErrInvalidInput,
			pipelineinternal.RunTypeManual,
			pipelineinternal.RunTypeScheduled,
			pipelineinternal.RunTypeCI,
		)
	}

	existing, err := FindByWorkflowRun(app, in.WorkflowID, in.RunID)
	if err == nil {
		if existing.GetString("owner") == in.OwnerID &&
			existing.GetString("pipeline") == in.PipelineID {
			return existing, nil
		}
		return nil, fmt.Errorf("%w: owner or pipeline mismatch", ErrConflict)
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	coll, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, fmt.Errorf("get pipeline_results collection: %w", err)
	}
	record := core.NewRecord(coll)
	record.Set("owner", in.OwnerID)
	record.Set("pipeline", in.PipelineID)
	record.Set("workflow_id", in.WorkflowID)
	record.Set("run_id", in.RunID)
	if deviceIDs := mobilerunner.NormalizeDeviceIDs(in.DeviceIDs); len(deviceIDs) > 0 {
		deviceRecordIDs := make([]string, 0, len(deviceIDs))
		for _, deviceID := range deviceIDs {
			device, err := canonify.Resolve(app, deviceID)
			if err != nil {
				return nil, fmt.Errorf("resolve device %s: %w", deviceID, err)
			}
			deviceRecordIDs = append(deviceRecordIDs, device.Id)
		}
		record.Set("devices", deviceRecordIDs)
	}
	if coll.Fields.GetByName("type") != nil {
		record.Set("type", runType)
	}
	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("save pipeline result: %w", err)
	}
	return record, nil
}

// StoreEvidence saves the credential well-knowns and presentation results of a run.
func StoreEvidence(
	app core.App,
	workflowID, runID string,
	credentialWellKnowns, presentationResults []map[string]any,
) error {
	if err := requireWorkflowRun(workflowID, runID); err != nil {
		return err
	}
	record, err := FindByWorkflowRun(app, workflowID, runID)
	if err != nil {
		return err
	}
	record.Set("credential_well_knowns", credentialWellKnowns)
	record.Set("presentation_results", presentationResults)
	if err := app.Save(record); err != nil {
		return fmt.Errorf("save pipeline evidence: %w", err)
	}
	return nil
}

// StoreReport saves the markdown report of a run under a sanitized filename.
func StoreReport(app core.App, workflowID, runID, filename, markdown string) error {
	if err := requireWorkflowRun(workflowID, runID); err != nil {
		return err
	}
	if strings.TrimSpace(markdown) == "" {
		return fmt.Errorf("%w: markdown is required", ErrInvalidInput)
	}
	record, err := FindByWorkflowRun(app, workflowID, runID)
	if err != nil {
		return err
	}
	file, err := filesystem.NewFileFromBytes([]byte(markdown), SanitizeReportFilename(filename))
	if err != nil {
		return fmt.Errorf("create report file: %w", err)
	}
	record.Set("report", []*filesystem.File{file})
	if err := app.Save(record); err != nil {
		return fmt.Errorf("save pipeline report: %w", err)
	}
	return nil
}

func requireWorkflowRun(workflowID, runID string) error {
	if strings.TrimSpace(workflowID) == "" || strings.TrimSpace(runID) == "" {
		return fmt.Errorf("%w: workflow_id and run_id are required", ErrInvalidInput)
	}
	return nil
}

var unsafeReportFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// SanitizeReportFilename returns a safe markdown filename for a run report:
// unsafe characters become "-", and the result always ends in ".md".
func SanitizeReportFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = "pipeline-report.md"
	}
	filename = unsafeReportFilenameChars.ReplaceAllString(filename, "-")
	filename = strings.Trim(filename, ".-_")
	if filename == "" {
		return "pipeline-report.md"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".md") {
		filename += ".md"
	}
	return filename
}
