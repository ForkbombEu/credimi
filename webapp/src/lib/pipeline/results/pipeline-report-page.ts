// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

const PIPELINE_RESULTS_FILE_PREFIX = '/api/files/pipeline_results/';

/**
 * Accepts a pipeline_results file URL and returns it on `origin`. Stored URLs
 * carry the backend App URL, which may name the same server differently from
 * the page (e.g. localhost vs a hostname), so only the path is kept: the page
 * always loads result files from its own server.
 */
export function validatePipelineReportUrl(reportUrl: string, origin: string): string | undefined {
	try {
		const parsed = new URL(reportUrl, origin);
		if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return undefined;
		if (!parsed.pathname.startsWith(PIPELINE_RESULTS_FILE_PREFIX)) return undefined;
		return new URL(parsed.pathname + parsed.search, origin).href;
	} catch {
		return undefined;
	}
}

export function buildPipelineReportPageHref(reportUrl: string): string {
	return `/pipeline-report?url=${encodeURIComponent(reportUrl)}`;
}

export function endPipelineReportBootLoading() {
	if (typeof document === 'undefined') return;
	document.documentElement.classList.remove('pipeline-report-loading');
}
