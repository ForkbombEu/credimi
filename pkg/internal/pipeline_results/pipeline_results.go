// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelineresults

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/pocketbase/pocketbase/core"
)

type PipelineResults struct {
	Video      string `json:"video,omitempty"`
	Screenshot string `json:"screenshot,omitempty"`
	Log        string `json:"log,omitempty"`
}

type PipelineExecutionArtifacts struct {
	Results []PipelineResults `json:"results"`
	Report  string            `json:"report,omitempty"`
}

func RegisterPipelineResultsHooks(app core.App) {
	app.OnRecordEnrich("pipeline_results").BindFunc(HandlePipelineResultsEnrich)
}

func HandlePipelineResultsEnrich(e *core.RecordEnrichEvent) error {
	artifacts := BuildPipelineExecutionArtifacts(e.App, e.Record)
	e.Record.WithCustomData(true)
	e.Record.Set("artifacts", artifacts)
	return e.Next()
}

func BuildPipelineExecutionArtifacts(app core.App, record *core.Record) PipelineExecutionArtifacts {
	results := ComputePipelineResultsFromRecord(app, record)
	if results == nil {
		results = []PipelineResults{}
	}

	return PipelineExecutionArtifacts{
		Results: results,
		Report:  ComputePipelineReportURLFromRecord(app, record),
	}
}

func ResolvePipelineExecutionArtifacts(
	app core.App,
	owner string,
	workflowID string,
	runID string,
) PipelineExecutionArtifacts {
	identifier := fmt.Sprintf(
		"%s/%s-%s",
		owner,
		canonify.CanonifyPlain(workflowID),
		canonify.CanonifyPlain(runID),
	)
	record, _ := canonify.Resolve(app, identifier)
	if record == nil {
		return PipelineExecutionArtifacts{Results: []PipelineResults{}}
	}

	return BuildPipelineExecutionArtifacts(app, record)
}

func ComputePipelineResultsFromRecord(app core.App, record *core.Record) []PipelineResults {
	if app == nil || record == nil {
		return nil
	}

	videos := record.GetStringSlice("video_results")
	screenshots := record.GetStringSlice("screenshots")
	if len(videos) == 0 || len(screenshots) == 0 {
		return nil
	}

	screenshotMap := make(map[string]string, len(screenshots))
	for _, name := range screenshots {
		if key, ok := baseArtifactKey(name, "_screenshot_"); ok {
			screenshotMap[key] = name
		}
	}

	logMap := make(map[string]pipelineResultFileRef)
	for _, field := range []string{"logcats", "ios_logstreams"} {
		for _, name := range record.GetStringSlice(field) {
			if key, ok := baseArtifactKey(name, "_logfile_"); ok {
				logMap[key] = pipelineResultFileRef{
					field: field,
					name:  name,
				}
			}
		}
	}

	results := make([]PipelineResults, 0, len(videos))
	for _, name := range videos {
		key, ok := baseArtifactKey(name, "_result_video_")
		if !ok {
			continue
		}

		screenshot, found := screenshotMap[key]
		if !found {
			continue
		}

		result := PipelineResults{
			Video: utils.JoinURL(
				app.Settings().Meta.AppURL,
				"api", "files", "pipeline_results",
				record.Id,
				record.GetString("video_results"),
				name,
			),
			Screenshot: utils.JoinURL(
				app.Settings().Meta.AppURL,
				"api", "files", "pipeline_results",
				record.Id,
				record.GetString("screenshots"),
				screenshot,
			),
		}
		if logFile, found := logMap[key]; found {
			result.Log = utils.JoinURL(
				app.Settings().Meta.AppURL,
				"api", "files", "pipeline_results",
				record.Id,
				record.GetString(logFile.field),
				logFile.name,
			)
		}

		results = append(results, result)
	}

	return results
}

func ComputePipelineReportURLFromRecord(app core.App, record *core.Record) string {
	if app == nil || record == nil {
		return ""
	}

	filename := record.GetString("report")
	if filename == "" {
		if reports := record.GetStringSlice("report"); len(reports) > 0 {
			filename = reports[0]
		}
	}
	if filename == "" {
		return ""
	}

	return utils.JoinURL(
		app.Settings().Meta.AppURL,
		"api", "files", "pipeline_results",
		record.Id,
		filename,
	)
}

type pipelineResultFileRef struct {
	field string
	name  string
}

func baseArtifactKey(filename, marker string) (string, bool) {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	idx := strings.LastIndex(name, marker)
	if idx == -1 {
		return "", false
	}

	return name[:idx], true
}
