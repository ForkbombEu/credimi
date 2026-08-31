// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/reportpdf"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/pocketbase/pocketbase/core"
)

const maximumFCAFImageReadBytes = 50 << 20

func generatePipelineFCAFReportPDF(
	ctx context.Context,
	app core.App,
	record *core.Record,
	rawJSON []byte,
) ([]byte, error) {
	var report engine.Report
	if err := json.Unmarshal(rawJSON, &report); err != nil {
		return nil, fmt.Errorf("decode FCAF report: %w", err)
	}
	if len(report.ExecutedTests) == 0 {
		return nil, fmt.Errorf("decode FCAF report: executed_tests is empty")
	}

	testIDs := make([]string, 0, len(report.ExecutedTests))
	for _, test := range report.ExecutedTests {
		if strings.TrimSpace(test.TestID) != "" {
			testIDs = append(testIDs, test.TestID)
		}
	}
	definitions, sources, warnings := reportpdf.LoadMaterials(testIDs)
	images, imageWarnings, err := loadPipelineFCAFReportImages(app, record, report)
	if err != nil {
		return nil, err
	}
	warnings = append(warnings, imageWarnings...)

	metadata, metadataWarnings := pipelineFCAFReportMetadata(app, record)
	warnings = append(warnings, metadataWarnings...)
	metadata.JSONFilename = record.GetString("fcaf_report")

	document := reportpdf.BuildDocument(reportpdf.Input{
		Report:      report,
		RawJSON:     rawJSON,
		Metadata:    metadata,
		Definitions: definitions,
		Sources:     sources,
		Images:      images,
		Warnings:    warnings,
	})
	pdf, err := reportpdf.Render(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("render FCAF report PDF: %w", err)
	}
	return pdf, nil
}

func loadPipelineFCAFReportImages(
	app core.App,
	record *core.Record,
	report engine.Report,
) ([]reportpdf.ImageAsset, []string, error) {
	if app == nil || record == nil {
		return nil, nil, fmt.Errorf("load FCAF report images: app and record are required")
	}

	storedFilenames := make([]string, 0)
	storedSet := map[string]struct{}{}
	for _, field := range []string{"maestro_screenshots", "screenshots"} {
		for _, filename := range record.GetStringSlice(field) {
			filename = strings.TrimSpace(filename)
			if filename == "" {
				continue
			}
			if _, found := storedSet[filename]; found {
				continue
			}
			storedSet[filename] = struct{}{}
			storedFilenames = append(storedFilenames, filename)
		}
	}
	if len(storedFilenames) == 0 && len(report.ExecutedTests) == 0 {
		return nil, nil, nil
	}

	warnings := []string{}
	seenWarnings := map[string]struct{}{}
	for _, execution := range report.ExecutedTests {
		for _, item := range execution.Evidence {
			for _, reference := range item.Visual {
				filename := reportpdf.ReferenceFilename(reference)
				if filename == "" {
					continue
				}
				if _, found := storedSet[filename]; found {
					continue
				}
				if _, warned := seenWarnings[filename]; warned {
					continue
				}
				seenWarnings[filename] = struct{}{}
				warnings = append(warnings, fmt.Sprintf(
					"visual evidence %s was not stored on this pipeline result",
					filename,
				))
			}
		}
	}

	if len(storedFilenames) == 0 {
		return nil, warnings, nil
	}
	fileSystem, err := app.NewFilesystem()
	if err != nil {
		return nil, warnings, fmt.Errorf("open PocketBase filesystem: %w", err)
	}
	defer fileSystem.Close()

	images := make([]reportpdf.ImageAsset, 0, len(storedFilenames))
	for _, filename := range storedFilenames {
		reader, err := fileSystem.GetFile(record.BaseFilesPath() + "/" + filename)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("read visual evidence %s: %v", filename, err))
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, maximumFCAFImageReadBytes+1))
		closeErr := reader.Close()
		if readErr != nil {
			warnings = append(
				warnings,
				fmt.Sprintf("read visual evidence %s: %v", filename, readErr),
			)
			continue
		}
		if closeErr != nil {
			warnings = append(
				warnings,
				fmt.Sprintf("close visual evidence %s: %v", filename, closeErr),
			)
			continue
		}
		prepared, err := reportpdf.PrepareImage(data)
		if err != nil {
			warnings = append(
				warnings,
				fmt.Sprintf("prepare visual evidence %s: %v", filename, err),
			)
			continue
		}

		images = append(images, reportpdf.ImageAsset{Filename: filename, Data: prepared})
	}

	return images, warnings, nil
}

func pipelineFCAFReportMetadata(
	app core.App,
	record *core.Record,
) (reportpdf.Metadata, []string) {
	metadata := reportpdf.Metadata{
		WorkflowID:  record.GetString("workflow_id"),
		RunID:       record.GetString("run_id"),
		GeneratedAt: record.GetDateTime("created").Time().UTC(),
	}
	warnings := []string{}
	if metadata.GeneratedAt.IsZero() {
		metadata.GeneratedAt = time.Now().UTC()
	}

	pipelineID := record.GetString("pipeline")
	if pipelineID != "" {
		pipelineRecord, err := app.FindRecordById("pipelines", pipelineID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("load pipeline metadata: %v", err))
		} else {
			metadata.PipelineName = resolvePipelineNameFromRecord(pipelineRecord, pipelineID)
			identifier, err := canonify.BuildPath(
				app,
				pipelineRecord,
				canonify.CanonifyPaths["pipelines"],
				"",
			)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("build pipeline identifier: %v", err))
			} else {
				metadata.PipelineIdentifier = strings.Trim(identifier, "/")
			}
			metadata.Runner = resolvePipelineRunner(app, pipelineRecord)
		}
	}

	ownerID := record.GetString("owner")
	if ownerID != "" {
		owner, err := app.FindRecordById("organizations", ownerID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("load organization metadata: %v", err))
		} else {
			metadata.OrganizationName = firstNonBlank(
				owner.GetString("name"),
				owner.GetString("canonified_name"),
			)
		}
	}
	return metadata, warnings
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func resolvePipelineRunner(app core.App, pipelineRecord *core.Record) reportpdf.RunnerInfo {
	wf, err := pipeline.ParseWorkflow(pipelineRecord.GetString("yaml"))
	if err != nil || wf.Runtime.GlobalRunnerID == "" {
		return reportpdf.RunnerInfo{}
	}
	runnerRecord, err := canonify.Resolve(app, canonify.NormalizePath(wf.Runtime.GlobalRunnerID))
	if err != nil {
		return reportpdf.RunnerInfo{}
	}
	return reportpdf.RunnerInfo{
		Name:   runnerRecord.GetString("name"),
		Type:   humanizeRunnerType(runnerRecord.GetString("type")),
		Serial: runnerRecord.GetString("serial"),
	}
}

func humanizeRunnerType(runnerType string) string {
	switch runnerType {
	case "android_emulator":
		return "Android emulator"
	case "android_phone":
		return "Android phone"
	case "ios_emulator", "ios_simulator":
		return "iOS simulator"
	case "ios_phone":
		return "iOS phone"
	default:
		return runnerType
	}
}
