// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

const PIPELINE_RESULTS_FILE_PREFIX = '/api/files/pipeline_results/';

export function validatePipelineReportUrl(reportUrl: string, origin: string): string | undefined {
	try {
		const parsed = new URL(reportUrl, origin);
		if (parsed.origin !== origin) return undefined;
		if (!parsed.pathname.startsWith(PIPELINE_RESULTS_FILE_PREFIX)) return undefined;
		return parsed.href;
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
