// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelineresults

import (
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const storeTestOrgID = "co35481b68u3zj3"

func newStoreTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	canonify.RegisterCanonifyHooks(app)
	return app
}

func createStorePipeline(t *testing.T, app *tests.TestApp, name string) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("owner", storeTestOrgID)
	record.Set("name", name)
	record.Set("description", "test-description")
	record.Set("yaml", "example-yaml-content")
	require.NoError(t, app.Save(record))
	return record
}

func createStoreDevice(t *testing.T, app *tests.TestApp) (*core.Record, string) {
	t.Helper()
	runnersColl, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(runnersColl)
	runner.Set("owner", storeTestOrgID)
	runner.Set("name", "Runner One")
	runner.Set("ip", "http://runner.test")
	runner.Set("type", "android_emulator")
	require.NoError(t, app.Save(runner))

	devicesColl, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(devicesColl)
	device.Set("owner", storeTestOrgID)
	device.Set("runner", runner.Id)
	device.Set("name", "Device One")
	device.Set("type", "android_emulator")
	device.Set("serial", "device-one-serial")
	require.NoError(t, app.Save(device))

	path, err := canonify.BuildPath(app, device, canonify.CanonifyPaths["mobile_devices"], "")
	require.NoError(t, err)
	return device, canonify.NormalizePath(path)
}

func TestCreate(t *testing.T) {
	for _, tc := range []struct {
		name        string
		runType     string
		withDevice  bool
		wantType    string
		wantInvalid bool
	}{
		{name: "CI run", runType: pipelineinternal.RunTypeCI, wantType: pipelineinternal.RunTypeCI},
		{name: "empty type defaults to manual", runType: "  ", wantType: pipelineinternal.RunTypeManual},
		{name: "invalid type", runType: "bogus", wantInvalid: true},
		{
			name:       "devices resolved into relation",
			runType:    pipelineinternal.RunTypeScheduled,
			withDevice: true,
			wantType:   pipelineinternal.RunTypeScheduled,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newStoreTestApp(t)
			pipeline := createStorePipeline(t, app, "pipeline123")

			in := CreateInput{
				OwnerID:    storeTestOrgID,
				PipelineID: pipeline.Id,
				WorkflowID: "workflow-xyz",
				RunID:      "run-001",
				RunType:    tc.runType,
			}
			var device *core.Record
			if tc.withDevice {
				var deviceID string
				device, deviceID = createStoreDevice(t, app)
				in.DeviceIDs = []string{" /" + deviceID + " ", deviceID}
			}

			record, err := Create(app, in)
			if tc.wantInvalid {
				require.ErrorIs(t, err, ErrInvalidInput)
				return
			}
			require.NoError(t, err)

			stored, err := FindByWorkflowRun(app, "workflow-xyz", "run-001")
			require.NoError(t, err)
			require.Equal(t, record.Id, stored.Id)
			require.Equal(t, storeTestOrgID, stored.GetString("owner"))
			require.Equal(t, pipeline.Id, stored.GetString("pipeline"))
			require.Equal(t, tc.wantType, stored.GetString("type"))
			if device != nil {
				require.Equal(t, []string{device.Id}, stored.GetStringSlice("devices"))
			} else {
				require.Empty(t, stored.GetStringSlice("devices"))
			}
		})
	}
}

func TestCreateUnknownDevice(t *testing.T) {
	app := newStoreTestApp(t)
	pipeline := createStorePipeline(t, app, "pipeline123")

	_, err := Create(app, CreateInput{
		OwnerID:    storeTestOrgID,
		PipelineID: pipeline.Id,
		WorkflowID: "workflow-xyz",
		RunID:      "run-001",
		DeviceIDs:  []string{"usera-s-organization/missing/device"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolve device")
}

func TestCreateIdempotent(t *testing.T) {
	app := newStoreTestApp(t)
	pipeline := createStorePipeline(t, app, "pipeline123")
	in := CreateInput{
		OwnerID:    storeTestOrgID,
		PipelineID: pipeline.Id,
		WorkflowID: "workflow-xyz",
		RunID:      "run-001",
	}

	first, err := Create(app, in)
	require.NoError(t, err)
	second, err := Create(app, in)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)

	records, err := app.FindRecordsByFilter(
		"pipeline_results",
		"workflow_id = {:workflow_id} && run_id = {:run_id}",
		"",
		-1,
		0,
		dbx.Params{"workflow_id": "workflow-xyz", "run_id": "run-001"},
	)
	require.NoError(t, err)
	require.Len(t, records, 1)
}

func TestCreateConflict(t *testing.T) {
	app := newStoreTestApp(t)
	pipelineA := createStorePipeline(t, app, "pipeline-a")
	pipelineB := createStorePipeline(t, app, "pipeline-b")
	in := CreateInput{
		OwnerID:    storeTestOrgID,
		PipelineID: pipelineA.Id,
		WorkflowID: "workflow-xyz",
		RunID:      "run-001",
	}
	_, err := Create(app, in)
	require.NoError(t, err)

	in.PipelineID = pipelineB.Id
	_, err = Create(app, in)
	require.ErrorIs(t, err, ErrConflict)
}

func TestFindByWorkflowRunNotFound(t *testing.T) {
	app := newStoreTestApp(t)

	_, err := FindByWorkflowRun(app, "workflow-missing", "run-missing")
	require.ErrorIs(t, err, ErrNotFound)
}

func createStoreResult(t *testing.T, app *tests.TestApp, workflowID, runID string) *core.Record {
	t.Helper()
	pipeline := createStorePipeline(t, app, "pipeline123")
	record, err := Create(app, CreateInput{
		OwnerID:    storeTestOrgID,
		PipelineID: pipeline.Id,
		WorkflowID: workflowID,
		RunID:      runID,
	})
	require.NoError(t, err)
	return record
}

func TestStoreEvidence(t *testing.T) {
	app := newStoreTestApp(t)
	result := createStoreResult(t, app, "workflow-evidence", "run-evidence")

	err := StoreEvidence(
		app,
		"workflow-evidence",
		"run-evidence",
		[]map[string]any{{
			"step_id":       "cred-step",
			"credential_id": "tenant/credential-1",
			"well_known":    map[string]any{"credential_issuer": "issuer-1"},
		}},
		[]map[string]any{{
			"step_id":     "vp-step",
			"use_case_id": "tenant/use-case-1",
			"result":      map[string]any{"format": "jwt"},
		}},
	)
	require.NoError(t, err)

	reloaded, err := app.FindRecordById("pipeline_results", result.Id)
	require.NoError(t, err)
	var credentialWellKnowns []map[string]any
	var presentationResults []map[string]any
	require.NoError(t, reloaded.UnmarshalJSONField("credential_well_knowns", &credentialWellKnowns))
	require.NoError(t, reloaded.UnmarshalJSONField("presentation_results", &presentationResults))
	require.Len(t, credentialWellKnowns, 1)
	require.Len(t, presentationResults, 1)
	require.Equal(t, "tenant/credential-1", credentialWellKnowns[0]["credential_id"])
}

func TestStoreEvidenceErrors(t *testing.T) {
	app := newStoreTestApp(t)

	require.ErrorIs(t, StoreEvidence(app, "", "run", nil, nil), ErrInvalidInput)
	require.ErrorIs(t, StoreEvidence(app, "workflow-missing", "run-missing", nil, nil), ErrNotFound)
}

func TestStoreReport(t *testing.T) {
	app := newStoreTestApp(t)
	result := createStoreResult(t, app, "workflow-report", "run-report")

	require.NoError(
		t,
		StoreReport(app, "workflow-report", "run-report", "../workflow report", "# Report\n\nBody"),
	)

	reloaded, err := app.FindRecordById("pipeline_results", result.Id)
	require.NoError(t, err)
	files := reloaded.GetStringSlice("report")
	require.Len(t, files, 1)
	require.Contains(t, files[0], "workflow_report")
	require.True(t, strings.HasSuffix(files[0], ".md"))
}

func TestStoreReportValidationErrors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		workflowID string
		runID      string
		markdown   string
		wantErr    error
	}{
		{name: "missing workflow identifiers", markdown: "# Report", wantErr: ErrInvalidInput},
		{
			name:       "missing markdown",
			workflowID: "workflow-missing",
			runID:      "run-missing",
			markdown:   " ",
			wantErr:    ErrInvalidInput,
		},
		{
			name:       "missing record",
			workflowID: "workflow-missing",
			runID:      "run-missing",
			markdown:   "# Report",
			wantErr:    ErrNotFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newStoreTestApp(t)
			err := StoreReport(app, tc.workflowID, tc.runID, "report.md", tc.markdown)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestSanitizeReportFilename(t *testing.T) {
	for input, want := range map[string]string{
		"":                   "pipeline-report.md",
		"../workflow report": "workflow-report.md",
		"workflow.md":        "workflow.md",
		"///":                "pipeline-report.md",
	} {
		require.Equal(t, want, sanitizeReportFilename(input), input)
	}
}
