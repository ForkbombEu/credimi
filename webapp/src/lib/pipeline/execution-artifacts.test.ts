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
			fcafReport: undefined
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
			fcafReport: undefined
		});
	});

	it('returns undefined when artifacts missing', () => {
		expect(fromEnrichedRecord({})).toBeUndefined();
	});
});
