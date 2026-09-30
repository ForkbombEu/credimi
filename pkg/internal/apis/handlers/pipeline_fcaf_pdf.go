// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/reportgeneration"
	"github.com/forkbombeu/credimi/pkg/fcaf/reportpdf"
	"github.com/pocketbase/pocketbase/core"
)

func generatePipelineFCAFReportPDF(
	ctx context.Context,
	app core.App,
	record *core.Record,
	rawJSON []byte,
) ([]byte, error) {
	return reportgeneration.GeneratePipelineFCAFReportPDF(ctx, app, record, rawJSON)
}

func loadPipelineFCAFReportImages(
	app core.App,
	record *core.Record,
	report engine.Report,
) ([]reportpdf.ImageAsset, []string, error) {
	return reportgeneration.LoadPipelineFCAFReportImages(app, record, report)
}
