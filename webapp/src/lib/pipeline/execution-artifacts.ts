// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type PipelineExecutionArtifacts = {
	results: Array<{ video: string; screenshot: string; log: string }>;
	maestro_screenshots?: string[];
	report?: string;
	fcafReport?: string;
	fcafReportPdf?: string;
};

export function fromApiSummary(summary: {
	results?: PipelineExecutionArtifacts['results'];
	maestro_screenshots?: string[];
	report?: string;
	fcaf_report?: string;
	fcaf_report_pdf?: string;
}): PipelineExecutionArtifacts | undefined {
	const hasResults = (summary.results?.length ?? 0) > 0;
	const hasReport = Boolean(summary.report);
	const hasFCAFReport = Boolean(summary.fcaf_report);
	const hasFCAFReportPdf = Boolean(summary.fcaf_report_pdf);
	if (!hasResults && !hasReport && !hasFCAFReport && !hasFCAFReportPdf) return undefined;
	return {
		results: summary.results ?? [],
		maestro_screenshots: summary.maestro_screenshots ?? [],
		report: summary.report,
		fcafReport: summary.fcaf_report,
		fcafReportPdf: summary.fcaf_report_pdf
	};
}

export function fromEnrichedRecord(record: {
	artifacts?: PipelineExecutionArtifacts & {
		fcaf_report?: string;
		fcaf_report_pdf?: string;
	};
}): PipelineExecutionArtifacts | undefined {
	if (!record.artifacts) return undefined;
	const { results, report } = record.artifacts;
	const fcafReport = record.artifacts.fcafReport ?? record.artifacts.fcaf_report;
	const fcafReportPdf = record.artifacts.fcafReportPdf ?? record.artifacts.fcaf_report_pdf;
	const hasResults = (results?.length ?? 0) > 0;
	const hasReport = Boolean(report);
	const hasFCAFReport = Boolean(fcafReport);
	const hasFCAFReportPdf = Boolean(fcafReportPdf);
	if (!hasResults && !hasReport && !hasFCAFReport && !hasFCAFReportPdf) return undefined;
	return {
		results: results ?? [],
		maestro_screenshots: record.artifacts.maestro_screenshots ?? [],
		report,
		fcafReport,
		fcafReportPdf
	};
}
