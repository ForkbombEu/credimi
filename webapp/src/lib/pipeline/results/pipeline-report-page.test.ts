// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { buildPipelineReportPageHref, validatePipelineReportUrl } from './pipeline-report-page';

const origin = 'https://app.test';

describe('validatePipelineReportUrl', () => {
	it('accepts same-origin pipeline_results file URLs', () => {
		expect(
			validatePipelineReportUrl(
				'https://app.test/api/files/pipeline_results/rec123/run_report.md',
				origin
			)
		).toBe('https://app.test/api/files/pipeline_results/rec123/run_report.md');
	});

	it('accepts relative pipeline_results file paths', () => {
		expect(
			validatePipelineReportUrl('/api/files/pipeline_results/rec123/run_report.md', origin)
		).toBe('https://app.test/api/files/pipeline_results/rec123/run_report.md');
	});

	it('rejects cross-origin URLs', () => {
		expect(
			validatePipelineReportUrl(
				'https://evil.test/api/files/pipeline_results/rec123/run_report.md',
				origin
			)
		).toBeUndefined();
	});

	it('rejects non-pipeline_results paths', () => {
		expect(
			validatePipelineReportUrl('https://app.test/api/files/users/rec123/avatar.png', origin)
		).toBeUndefined();
	});

	it('rejects invalid URLs', () => {
		expect(validatePipelineReportUrl('not-a-url', origin)).toBeUndefined();
	});
});

describe('buildPipelineReportPageHref', () => {
	it('builds a query param link', () => {
		expect(
			buildPipelineReportPageHref(
				'https://app.test/api/files/pipeline_results/rec123/run_report.md'
			)
		).toBe(
			'/pipeline-report?url=' +
				encodeURIComponent('https://app.test/api/files/pipeline_results/rec123/run_report.md')
		);
	});
});
