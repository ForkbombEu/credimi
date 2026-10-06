// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"io"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const (
	pipelineResultsTestOrgID        = "co35481b68u3zj3"
	pipelineResultsTestOrgNamespace = "usera-s-organization"
	pipelineResultsTestAppURL       = "https://credimi.test"
)

// newPipelineResultsTestApp returns a test app with canonify hooks and the
// Settings app URL set.
func newPipelineResultsTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	canonify.RegisterCanonifyHooks(app)
	app.Settings().Meta.AppURL = pipelineResultsTestAppURL
	app.Settings().Meta.AppName = "Credimi"
	return app
}

// createTestPipeline stores a pipeline of the test organization and returns it
// with its canonified identifier.
func createTestPipeline(t *testing.T, app core.App, name string) (*core.Record, string) {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("owner", pipelineResultsTestOrgID)
	record.Set("name", name)
	record.Set("description", "test-description")
	record.Set("yaml", "name: "+name+"\nsteps: []\n")
	require.NoError(t, app.Save(record))

	path, err := canonify.BuildPath(app, record, canonify.CanonifyPaths["pipelines"], "")
	require.NoError(t, err)
	return record, canonify.NormalizePath(path)
}

// createTestPipelineResult stores the pipeline result of a run of a new pipeline.
func createTestPipelineResult(
	t *testing.T,
	app core.App,
	workflowID string,
	runID string,
) *core.Record {
	t.Helper()
	pipeline, _ := createTestPipeline(t, app, "pipeline-"+workflowID)
	record, err := pipelineresults.Create(app, pipelineresults.CreateInput{
		OwnerID:    pipelineResultsTestOrgID,
		PipelineID: pipeline.Id,
		WorkflowID: workflowID,
		RunID:      runID,
	})
	require.NoError(t, err)
	return record
}

// readPipelineResultFile returns the content of the file stored in field.
func readPipelineResultFile(t *testing.T, app core.App, record *core.Record, field string) []byte {
	t.Helper()
	fileSystem, err := app.NewFilesystem()
	require.NoError(t, err)
	defer fileSystem.Close()
	reader, err := fileSystem.GetReader(record.BaseFilesPath() + "/" + record.GetString(field))
	require.NoError(t, err)
	defer reader.Close()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	return data
}

// ensureFCAFReportFields adds the FCAF report file fields to pipeline_results
// when the test schema lacks them.
func ensureFCAFReportFields(t *testing.T, app core.App) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	if collection.Fields.GetByName("fcaf_report") == nil {
		collection.Fields.Add(&core.FileField{Name: "fcaf_report", MaxSelect: 1})
	}
	if collection.Fields.GetByName("fcaf_report_pdf") == nil {
		collection.Fields.Add(&core.FileField{Name: "fcaf_report_pdf", MaxSelect: 1})
	}
	require.NoError(t, app.Save(collection))
}
