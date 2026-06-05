// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type PipelineExecutionArtifacts = {
	results: Array<{ video: string; screenshot: string; log: string }>;
	report?: string;
};

export function fromApiSummary(summary: {
	results?: PipelineExecutionArtifacts['results'];
	report?: string;
}): PipelineExecutionArtifacts | undefined {
	const hasResults = (summary.results?.length ?? 0) > 0;
	const hasReport = Boolean(summary.report);
	if (!hasResults && !hasReport) return undefined;
	return {
		results: summary.results ?? [],
		report: summary.report
	};
}

export function fromEnrichedRecord(record: {
	artifacts?: PipelineExecutionArtifacts;
}): PipelineExecutionArtifacts | undefined {
	if (!record.artifacts) return undefined;
	const { results, report } = record.artifacts;
	const hasResults = (results?.length ?? 0) > 0;
	const hasReport = Boolean(report);
	if (!hasResults && !hasReport) return undefined;
	return { results: results ?? [], report };
}
