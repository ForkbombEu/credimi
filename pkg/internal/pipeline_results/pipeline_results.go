// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelineresults

import (
	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterPipelineResultsHooks(app core.App) {
	app.OnRecordEnrich("pipeline_results").BindFunc(HandlePipelineResultsEnrich)
}

func HandlePipelineResultsEnrich(e *core.RecordEnrichEvent) error {
	artifacts := handlers.BuildPipelineExecutionArtifacts(e.App, e.Record)
	e.Record.WithCustomData(true)
	e.Record.Set("artifacts", artifacts)
	return e.Next()
}
