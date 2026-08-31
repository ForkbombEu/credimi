// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { fromApiSummary, fromEnrichedRecord } from './execution-artifacts';

describe('fromApiSummary', () => {
	it('returns undefined when no results and no report', () => {
		expect(fromApiSummary({})).toBeUndefined();
	});

	it('maps results and report', () => {
		expect(
			fromApiSummary({
				results: [{ video: 'v', screenshot: 's', log: 'l' }],
				report: 'https://app/r.md'
			})
		).toEqual({
			results: [{ video: 'v', screenshot: 's', log: 'l' }],
			maestro_screenshots: [],
			report: 'https://app/r.md',
			fcafReport: undefined,
			fcafReportPdf: undefined
		});
	});

	it('maps FCAF JSON and PDF reports', () => {
		expect(
			fromApiSummary({
				fcaf_report: 'https://app/fcaf.json',
				fcaf_report_pdf: 'https://app/fcaf.pdf'
			})
		).toEqual({
			results: [],
			maestro_screenshots: [],
			report: undefined,
			fcafReport: 'https://app/fcaf.json',
			fcafReportPdf: 'https://app/fcaf.pdf'
		});
	});
});

describe('fromEnrichedRecord', () => {
	it('returns artifacts when present', () => {
		expect(
			fromEnrichedRecord({
				artifacts: { results: [{ video: 'v', screenshot: 's', log: 'l' }] }
			})
		).toEqual({
			results: [{ video: 'v', screenshot: 's', log: 'l' }],
			maestro_screenshots: [],
			report: undefined,
			fcafReport: undefined,
			fcafReportPdf: undefined
		});
	});

	it('normalizes snake-case enriched FCAF artifacts', () => {
		expect(
			fromEnrichedRecord({
				artifacts: {
					results: [],
					fcaf_report: 'https://app/fcaf.json',
					fcaf_report_pdf: 'https://app/fcaf.pdf'
				}
			})
		).toEqual({
			results: [],
			maestro_screenshots: [],
			report: undefined,
			fcafReport: 'https://app/fcaf.json',
			fcafReportPdf: 'https://app/fcaf.pdf'
		});
	});

	it('returns undefined when artifacts missing', () => {
		expect(fromEnrichedRecord({})).toBeUndefined();
	});
});
