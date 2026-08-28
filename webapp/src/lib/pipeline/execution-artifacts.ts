// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type PipelineExecutionArtifacts = {
	results: Array<{ video: string; screenshot: string; log: string }>;
	maestro_screenshots?: string[];
	report?: string;
	fcafReport?: string;
};

export function fromApiSummary(summary: {
	results?: PipelineExecutionArtifacts['results'];
	maestro_screenshots?: string[];
	report?: string;
	fcaf_report?: string;
}): PipelineExecutionArtifacts | undefined {
	const hasResults = (summary.results?.length ?? 0) > 0;
	const hasReport = Boolean(summary.report);
	const hasFCAFReport = Boolean(summary.fcaf_report);
	if (!hasResults && !hasReport && !hasFCAFReport) return undefined;
	return {
		results: summary.results ?? [],
		maestro_screenshots: summary.maestro_screenshots ?? [],
		report: summary.report,
		fcafReport: summary.fcaf_report
	};
}

export function fromEnrichedRecord(record: {
	artifacts?: PipelineExecutionArtifacts;
}): PipelineExecutionArtifacts | undefined {
	if (!record.artifacts) return undefined;
	const { results, report, fcafReport } = record.artifacts;
	const hasResults = (results?.length ?? 0) > 0;
	const hasReport = Boolean(report);
	const hasFCAFReport = Boolean(fcafReport);
	if (!hasResults && !hasReport && !hasFCAFReport) return undefined;
	return {
		results: results ?? [],
		maestro_screenshots: record.artifacts.maestro_screenshots ?? [],
		report,
		fcafReport
	};
}
