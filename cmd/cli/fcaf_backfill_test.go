// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

func TestFCAFReportNeedsPresentationBackfill(t *testing.T) {
	tests := []struct {
		name   string
		report engine.Report
		want   bool
	}{
		{
			name: "missing presentation",
			want: true,
		},
		{
			name: "empty presentation is already built",
			report: engine.Report{
				Presentation: &engine.Presentation{},
			},
			want: false,
		},
		{
			name: "populated presentation is already built",
			report: engine.Report{
				Presentation: &engine.Presentation{
					Deeplink: "openid4vp://authorize",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, fcafReportNeedsPresentationBackfill(tt.report))
		})
	}
}

func TestBackfillFCAFPresentationsUpdatesOnce(t *testing.T) {
	app, err := tests.NewTestApp("../../test_pb_data")
	require.NoError(t, err)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	if collection.Fields.GetByName("fcaf_report") == nil {
		collection.Fields.Add(&core.FileField{Name: "fcaf_report", MaxSelect: 1})
		require.NoError(t, app.Save(collection))
	}

	organization, err := app.FindFirstRecordByFilter(
		"organizations",
		"name = {:name}",
		map[string]any{"name": "userA's organization"},
	)
	require.NoError(t, err)
	pipelineCollection, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelineCollection)
	pipelineRecord.Set("owner", organization.Id)
	pipelineRecord.Set("name", "FCAF presentation backfill test")
	pipelineRecord.Set("canonified_name", "fcaf-presentation-backfill-test")
	pipelineRecord.Set("description", "FCAF presentation backfill test")
	pipelineRecord.Set("yaml", "name: FCAF presentation backfill test\nsteps: []\n")
	require.NoError(t, app.Save(pipelineRecord))

	reportJSON, err := json.Marshal(engine.Report{
		Suite:  "wallet_solution/relying_party",
		Status: "passed",
	})
	require.NoError(t, err)
	reportFile, err := filesystem.NewFileFromBytes(reportJSON, "historical-fcaf-report.json")
	require.NoError(t, err)

	record := core.NewRecord(collection)
	record.Set("owner", organization.Id)
	record.Set("pipeline", pipelineRecord.Id)
	record.Set("workflow_id", "workflow-backfill-presentation")
	record.Set("run_id", "run-backfill-presentation")
	record.Set("fcaf_report", []*filesystem.File{reportFile})
	require.NoError(t, app.Save(record))

	var out bytes.Buffer
	var errOut bytes.Buffer
	summary, err := backfillFCAFPresentations(
		context.Background(),
		app,
		fcafPresentationBackfillOptions{skipPDF: true},
		&out,
		&errOut,
	)
	require.NoError(t, err)
	require.Equal(t, fcafPresentationBackfillSummary{
		scanned: 1,
		updated: 1,
	}, summary)
	require.Empty(t, errOut.String())

	reloaded, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	fileSystem, err := app.NewFilesystem()
	require.NoError(t, err)
	defer fileSystem.Close()
	enrichedJSON, err := readFCAFReport(
		fileSystem,
		reloaded,
		reloaded.GetString("fcaf_report"),
	)
	require.NoError(t, err)
	var enrichedReport engine.Report
	require.NoError(t, json.Unmarshal(enrichedJSON, &enrichedReport))
	require.NotNil(t, enrichedReport.Presentation)

	out.Reset()
	summary, err = backfillFCAFPresentations(
		context.Background(),
		app,
		fcafPresentationBackfillOptions{skipPDF: true},
		&out,
		&errOut,
	)
	require.NoError(t, err)
	require.Equal(t, fcafPresentationBackfillSummary{
		scanned: 1,
		skipped: 1,
	}, summary)
	require.Empty(t, out.String())
	require.Empty(t, errOut.String())
}
