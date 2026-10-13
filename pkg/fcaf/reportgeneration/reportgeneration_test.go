// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportgeneration

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/reportpdf"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

const (
	testDataDir    = "../../../test_pb_data/"
	testOrgID      = "co35481b68u3zj3"
	testReportTest = "WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"
)

func TestEnrichReportJSONRejectsNonObjectRoots(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawJSON string
	}{
		{name: "null", rawJSON: "null"},
		{name: "array", rawJSON: `[]`},
		{name: "string", rawJSON: `"report"`},
		{name: "number", rawJSON: `1`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			enriched, report, err := EnrichReportJSON(nil, nil, []byte(tt.rawJSON))
			require.Error(t, err)
			require.Nil(t, enriched)
			require.Nil(t, report)
			require.Contains(t, err.Error(), "decode FCAF report object")
		})
	}
}

func TestEnrichReportJSONAttachesPresentationToObjectRoot(t *testing.T) {
	t.Parallel()

	enriched, report, err := EnrichReportJSON(nil, nil, []byte(`{"status":"passed"}`))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.Presentation)
	require.Contains(t, string(enriched), `"presentation"`)
	require.Contains(t, string(enriched), `"status": "passed"`)
}

func TestEnrichReportJSONRejectsMistypedReportFields(t *testing.T) {
	t.Parallel()

	enriched, report, err := EnrichReportJSON(nil, nil, []byte(`{"executed_tests":"none"}`))

	require.ErrorContains(t, err, "decode FCAF report:")
	require.NotContains(t, err.Error(), "report object")
	require.Nil(t, enriched)
	require.Nil(t, report)
}

func TestEnrichReportJSONListsStoredMaestroScreenshots(t *testing.T) {
	app := newReportTestApp(t)
	record := createReportPipelineResult(t, app, reportPipelineResultOptions{
		maestroScreenshots: map[string][]byte{"wallet_home.png": testPNG(t)},
	})
	stored := record.GetStringSlice("maestro_screenshots")
	require.Len(t, stored, 1)

	enriched, report, err := EnrichReportJSON(
		app,
		record,
		[]byte(`{"status":"passed","custom_field":{"kept":true}}`),
	)

	require.NoError(t, err)
	require.Len(t, report.Presentation.Screenshots, 1)
	require.Contains(t, report.Presentation.Screenshots[0].URL, record.Id+"/"+stored[0])

	var root map[string]any
	require.NoError(t, json.Unmarshal(enriched, &root))
	require.Equal(t, map[string]any{"kept": true}, root["custom_field"])
	presentation, ok := root["presentation"].(map[string]any)
	require.True(t, ok)
	screenshots, ok := presentation["screenshots"].([]any)
	require.True(t, ok)
	require.Len(t, screenshots, 1)
}

func TestAttachPresentationWithoutScreenshotSource(t *testing.T) {
	t.Parallel()

	report := &engine.Report{}
	AttachPresentation(nil, nil, report)

	require.Equal(
		t,
		&engine.Presentation{SummaryFilters: []engine.PresentationFilter{}},
		report.Presentation,
	)
}

func TestLoadPipelineFCAFReportImagesRequiresAppAndRecord(t *testing.T) {
	t.Parallel()

	images, warnings, err := LoadPipelineFCAFReportImages(nil, nil, engine.Report{})

	require.EqualError(t, err, "load FCAF report images: app and record are required")
	require.Nil(t, images)
	require.Nil(t, warnings)
}

func TestLoadPipelineFCAFReportImagesWithoutScreenshotsOrTests(t *testing.T) {
	app := newReportTestApp(t)
	record := createReportPipelineResult(t, app, reportPipelineResultOptions{})

	images, warnings, err := LoadPipelineFCAFReportImages(app, record, engine.Report{})

	require.NoError(t, err)
	require.Nil(t, images)
	require.Nil(t, warnings)
}

func TestLoadPipelineFCAFReportImagesWarnsAboutUnstoredVisualEvidence(t *testing.T) {
	app := newReportTestApp(t)
	record := createReportPipelineResult(t, app, reportPipelineResultOptions{})
	report := engine.Report{ExecutedTests: []engine.ExecutedTest{
		{
			TestID: "first",
			Evidence: []engine.ExecutedEvidence{{
				Name: "wallet_screen",
				Visual: []string{
					"https://app.test/api/files/pipeline_results/" + record.Id + "/missing.png?token=x",
					"",
				},
			}},
		},
		{
			TestID: "second",
			Evidence: []engine.ExecutedEvidence{{
				Name:   "wallet_screen",
				Visual: []string{"missing.png", "other.png"},
			}},
		},
	}}

	images, warnings, err := LoadPipelineFCAFReportImages(app, record, report)

	require.NoError(t, err)
	require.Nil(t, images)
	require.Equal(t, []string{
		"visual evidence missing.png was not stored on this pipeline result",
		"visual evidence other.png was not stored on this pipeline result",
	}, warnings)
}

func TestLoadPipelineFCAFReportImagesPreparesStoredScreenshots(t *testing.T) {
	app := newReportTestApp(t)
	record := createReportPipelineResult(t, app, reportPipelineResultOptions{
		maestroScreenshots: map[string][]byte{
			"wallet_home.png": testPNG(t),
			"corrupted.png":   []byte("not an image"),
		},
		screenshots: map[string][]byte{"final.png": testPNG(t)},
	})
	maestro := record.GetStringSlice("maestro_screenshots")
	final := record.GetStringSlice("screenshots")
	require.Len(t, maestro, 2)
	require.Len(t, final, 1)
	var corrupted, walletHome string
	for _, filename := range maestro {
		if strings.HasPrefix(filename, "corrupted") {
			corrupted = filename
		} else {
			walletHome = filename
		}
	}

	// A filename listed in both fields and one whose file is gone from storage.
	record.Set("screenshots", append(final, walletHome, " ", "deleted.png"))
	report := engine.Report{ExecutedTests: []engine.ExecutedTest{{
		TestID: testReportTest,
		Evidence: []engine.ExecutedEvidence{{
			Name:   "wallet_screen",
			Visual: []string{walletHome, "https://app.test/files/" + final[0]},
		}},
	}}}

	images, warnings, err := LoadPipelineFCAFReportImages(app, record, report)

	require.NoError(t, err)
	filenames := make([]string, 0, len(images))
	for _, image := range images {
		filenames = append(filenames, image.Filename)
		require.True(
			t,
			bytes.HasPrefix(image.Data, []byte{0xFF, 0xD8}),
			"%s must be converted to JPEG",
			image.Filename,
		)
	}
	require.Equal(t, []string{walletHome, final[0]}, filenames)
	require.Len(t, warnings, 2)
	require.Contains(t, warnings[0], "prepare visual evidence "+corrupted+":")
	require.Contains(t, warnings[1], "read visual evidence deleted.png:")
}

func TestPipelineFCAFReportMetadataResolvesPipelineOrganizationAndRunner(t *testing.T) {
	app := newReportTestApp(t)
	devicePath := createReportDevice(t, app)
	pipeline := createReportPipeline(
		t,
		app,
		"stored-pipeline-name",
		"name: FCAF complete validation\nruntime:\n  global_device_id: "+devicePath+"\nsteps: []\n",
	)
	record := createReportPipelineResult(t, app, reportPipelineResultOptions{pipeline: pipeline})
	organization, err := app.FindRecordById("organizations", testOrgID)
	require.NoError(t, err)

	metadata, warnings := pipelineFCAFReportMetadata(app, record)

	require.Empty(t, warnings)
	require.Equal(t, "FCAF complete validation", metadata.PipelineName)
	require.Equal(
		t,
		organization.GetString("canonified_name")+"/"+pipeline.GetString("canonified_name"),
		metadata.PipelineIdentifier,
	)
	require.Equal(t, organization.GetString("name"), metadata.OrganizationName)
	require.Equal(t, "Device One", metadata.Runner.Name)
	require.Equal(t, "Android emulator", metadata.Runner.Type)
	require.Equal(t, "device-one-serial", metadata.Runner.Serial)
	require.Equal(t, record.GetString("workflow_id"), metadata.WorkflowID)
	require.Equal(t, record.GetString("run_id"), metadata.RunID)
	require.Equal(t, record.GetDateTime("created").Time().UTC(), metadata.GeneratedAt)
}

func TestPipelineFCAFReportMetadataWarnsAboutMissingRelations(t *testing.T) {
	app := newReportTestApp(t)
	collection, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("pipeline", "missingpipeline")
	record.Set("owner", "missingowner01")
	record.Set("workflow_id", "workflow-1")

	before := time.Now().UTC()
	metadata, warnings := pipelineFCAFReportMetadata(app, record)

	require.Len(t, warnings, 2)
	require.Contains(t, warnings[0], "load pipeline metadata:")
	require.Contains(t, warnings[1], "load organization metadata:")
	require.Empty(t, metadata.PipelineName)
	require.Empty(t, metadata.OrganizationName)
	require.Equal(t, "workflow-1", metadata.WorkflowID)
	require.False(
		t,
		metadata.GeneratedAt.Before(before),
		"an unsaved record falls back to the generation time",
	)
}

func TestPipelineFCAFReportMetadataFallsBackToOrganizationCanonifiedName(t *testing.T) {
	app := newReportTestApp(t)
	organization, err := app.FindRecordById("organizations", testOrgID)
	require.NoError(t, err)
	canonifiedName := organization.GetString("canonified_name")
	require.NotEmpty(t, canonifiedName)
	organization.Set("name", "   ")
	organization.Set("canonified_name", canonifiedName)
	require.NoError(t, app.UnsafeWithoutHooks().SaveNoValidate(organization))
	record := createReportPipelineResult(t, app, reportPipelineResultOptions{})

	metadata, warnings := pipelineFCAFReportMetadata(app, record)

	require.Empty(t, warnings)
	require.Equal(t, canonifiedName, metadata.OrganizationName)
	require.Equal(
		t,
		reportpdf.RunnerInfo{},
		metadata.Runner,
		"a pipeline without a global device has no runner",
	)
}

func TestResolvePipelineRunnerIgnoresUnresolvableDevices(t *testing.T) {
	app := newReportTestApp(t)

	tests := []struct {
		name string
		yaml string
	}{
		{name: "invalid yaml", yaml: "name: [unterminated"},
		{name: "no global device", yaml: "name: pipeline\nsteps: []\n"},
		{
			name: "unknown device",
			yaml: "name: pipeline\nruntime:\n  global_device_id: org/runner/missing-device\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			collection, err := app.FindCollectionByNameOrId("pipelines")
			require.NoError(t, err)
			pipeline := core.NewRecord(collection)
			pipeline.Set("yaml", tc.yaml)

			require.Equal(t, reportpdf.RunnerInfo{}, resolvePipelineRunner(app, pipeline))
		})
	}
}

func TestHumanizeRunnerType(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"android_emulator": "Android emulator",
		"android_phone":    "Android phone",
		"ios_emulator":     "iOS simulator",
		"ios_simulator":    "iOS simulator",
		"ios_phone":        "iOS phone",
		"desktop":          "desktop",
		"":                 "",
	}
	for runnerType, want := range tests {
		require.Equal(t, want, humanizeRunnerType(runnerType), runnerType)
	}
}

func TestResolvePipelineNameFromRecord(t *testing.T) {
	t.Parallel()

	collection := core.NewBaseCollection("pipelines")
	collection.Fields.Add(&core.TextField{Name: "name"}, &core.TextField{Name: "yaml"})
	newPipeline := func(name, yaml string) *core.Record {
		record := core.NewRecord(collection)
		record.Set("name", name)
		record.Set("yaml", yaml)
		return record
	}

	tests := []struct {
		name     string
		record   *core.Record
		fallback string
		want     string
	}{
		{name: "nil record uses fallback", fallback: " pipeline-id ", want: "pipeline-id"},
		{name: "blank fallback uses default", fallback: "  ", want: "pipeline-run"},
		{
			name:     "workflow name wins over record name",
			record:   newPipeline("Stored name", "name: '  Workflow name  '\n"),
			fallback: "pipeline-id",
			want:     "Workflow name",
		},
		{
			name:     "invalid yaml uses record name",
			record:   newPipeline(" Stored name ", "name: [unterminated"),
			fallback: "pipeline-id",
			want:     "Stored name",
		},
		{
			name:     "blank workflow name uses record name",
			record:   newPipeline("Stored name", "steps: []\n"),
			fallback: "pipeline-id",
			want:     "Stored name",
		},
		{
			name:     "no names uses fallback",
			record:   newPipeline("", ""),
			fallback: "pipeline-id",
			want:     "pipeline-id",
		},
	}

	for _, tc := range tests {
		require.Equal(
			t,
			tc.want,
			resolvePipelineNameFromRecord(tc.record, tc.fallback),
			tc.name,
		)
	}
}

func TestGeneratePipelineFCAFReportPDFRejectsUnusableReports(t *testing.T) {
	app := newReportTestApp(t)
	record := createReportPipelineResult(t, app, reportPipelineResultOptions{})

	tests := []struct {
		name    string
		rawJSON string
		wantErr string
	}{
		{name: "invalid json", rawJSON: `{`, wantErr: "decode FCAF report:"},
		{
			name:    "no executed tests",
			rawJSON: `{"status":"passed","executed_tests":[]}`,
			wantErr: "decode FCAF report: executed_tests is empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pdf, err := GeneratePipelineFCAFReportPDF(
				context.Background(),
				app,
				record,
				[]byte(tc.rawJSON),
			)
			require.ErrorContains(t, err, tc.wantErr)
			require.Nil(t, pdf)
		})
	}

	_, err := GeneratePipelineFCAFReportPDF(
		context.Background(),
		nil,
		nil,
		[]byte(`{"executed_tests":[{"test_id":"`+testReportTest+`","status":"passed"}]}`),
	)
	require.EqualError(t, err, "load FCAF report images: app and record are required")
}

func TestGeneratePipelineFCAFReportPDFEmbedsStoredScreenshots(t *testing.T) {
	app := newReportTestApp(t)
	pipeline := createReportPipeline(t, app, "pdf-pipeline", "name: FCAF PDF\nsteps: []\n")
	withScreenshot := createReportPipelineResult(t, app, reportPipelineResultOptions{
		pipeline:           pipeline,
		maestroScreenshots: map[string][]byte{"wallet_home.png": testPNG(t)},
	})
	withoutScreenshot := createReportPipelineResult(t, app, reportPipelineResultOptions{
		pipeline: pipeline,
	})
	screenshot := withScreenshot.GetStringSlice("maestro_screenshots")[0]
	rawJSON := []byte(`{
		"suite": "wallet_solution/relying_party",
		"status": "passed",
		"summary": {"pass": 1},
		"executed_tests": [{
			"test_id": "` + testReportTest + `",
			"status": "passed",
			"evidence": [{"name": "wallet_screen", "visual": ["` + screenshot + `"]}]
		}]
	}`)

	withImage, err := GeneratePipelineFCAFReportPDF(
		context.Background(),
		app,
		withScreenshot,
		rawJSON,
	)
	require.NoError(t, err)
	withoutImage, err := GeneratePipelineFCAFReportPDF(
		context.Background(),
		app,
		withoutScreenshot,
		rawJSON,
	)
	require.NoError(t, err)

	require.True(t, bytes.HasPrefix(withImage, []byte("%PDF-")))
	require.True(t, bytes.HasPrefix(withoutImage, []byte("%PDF-")))
	require.Equal(
		t,
		bytes.Count(withoutImage, []byte("/Subtype /Image"))+1,
		bytes.Count(withImage, []byte("/Subtype /Image")),
		"the stored screenshot is embedded as one additional image",
	)
}

func TestGeneratePipelineFCAFReportPDFHonorsCanceledContext(t *testing.T) {
	app := newReportTestApp(t)
	record := createReportPipelineResult(t, app, reportPipelineResultOptions{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	pdf, err := GeneratePipelineFCAFReportPDF(
		ctx,
		app,
		record,
		[]byte(`{"executed_tests":[{"test_id":"`+testReportTest+`","status":"passed"}]}`),
	)

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "render FCAF report PDF:")
	require.Nil(t, pdf)
}

type reportPipelineResultOptions struct {
	pipeline           *core.Record
	maestroScreenshots map[string][]byte
	screenshots        map[string][]byte
}

func newReportTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	canonify.RegisterCanonifyHooks(app)
	return app
}

func createReportPipeline(t *testing.T, app *tests.TestApp, name, yaml string) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("owner", testOrgID)
	record.Set("name", name)
	record.Set("description", "FCAF report generation test")
	record.Set("yaml", yaml)
	require.NoError(t, app.Save(record))
	return record
}

func createReportDevice(t *testing.T, app *tests.TestApp) string {
	t.Helper()
	runnersCollection, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(runnersCollection)
	runner.Set("owner", testOrgID)
	runner.Set("name", "Runner One")
	runner.Set("ip", "http://runner.test")
	runner.Set("type", "android_emulator")
	require.NoError(t, app.Save(runner))

	devicesCollection, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(devicesCollection)
	device.Set("owner", testOrgID)
	device.Set("runner", runner.Id)
	device.Set("name", "Device One")
	device.Set("type", "android_emulator")
	device.Set("serial", "device-one-serial")
	require.NoError(t, app.Save(device))

	path, err := canonify.BuildPath(app, device, canonify.CanonifyPaths["mobile_devices"], "")
	require.NoError(t, err)
	return canonify.NormalizePath(path)
}

func createReportPipelineResult(
	t *testing.T,
	app *tests.TestApp,
	options reportPipelineResultOptions,
) *core.Record {
	t.Helper()
	pipeline := options.pipeline
	if pipeline == nil {
		pipeline = createReportPipeline(t, app, "report-pipeline", "steps: []\n")
	}
	collection, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("owner", testOrgID)
	record.Set("pipeline", pipeline.Id)
	record.Set("workflow_id", "workflow-report")
	record.Set("run_id", "run-"+pipeline.Id)
	record.Set("maestro_screenshots", testFiles(t, options.maestroScreenshots))
	record.Set("screenshots", testFiles(t, options.screenshots))
	require.NoError(t, app.Save(record))

	saved, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	return saved
}

func testFiles(t *testing.T, contents map[string][]byte) []*filesystem.File {
	t.Helper()
	files := make([]*filesystem.File, 0, len(contents))
	for name, data := range contents {
		file, err := filesystem.NewFileFromBytes(data, name)
		require.NoError(t, err)
		files = append(files, file)
	}
	return files
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{R: 49, G: 4, B: 255, A: 255})
		}
	}
	var output bytes.Buffer
	require.NoError(t, png.Encode(&output, img))
	return output.Bytes()
}
