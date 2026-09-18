// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	collectScreenshots,
	dedupeBurstScreenshots,
	evidenceDeeplink,
	evidenceScreenshotUrls,
	groupExecutedTests,
	prepareReportDisplay,
	screenshotLabel,
	screenshotsForTest,
	sourceUrl,
	statusIsFailed,
	statusIsPassed,
	testMatchesSearch,
	type Report,
	type TestResult
} from './report';

describe('status helpers', () => {
	it('treats pass prefixes as passed', () => {
		expect(statusIsPassed('passed')).toBe(true);
		expect(statusIsPassed('pass')).toBe(true);
		expect(statusIsPassed('failed')).toBe(false);
	});

	it('recognizes failed statuses', () => {
		expect(statusIsFailed('failed')).toBe(true);
		expect(statusIsFailed('error')).toBe(true);
		expect(statusIsFailed('passed')).toBe(false);
	});
});

describe('screenshot helpers', () => {
	it('extracts image urls from nested evidence', () => {
		expect(
			evidenceScreenshotUrls({
				a: 'https://example.com/a.png',
				b: [{ c: 'https://example.com/b.jpg?x=1' }, 'skip.txt']
			})
		).toEqual(['https://example.com/a.png', 'https://example.com/b.jpg?x=1']);
	});

	it('humanizes screenshot filenames', () => {
		expect(screenshotLabel('https://x/foo_bar-baz.png')).toBe('foo bar baz');
	});

	it('keeps the last frame of a maestro burst', () => {
		const shots = [
			{ url: '/run_screenshot_1_action_Tap.yaml1.png', label: 'a' },
			{ url: '/run_screenshot_2_action_Tap.yaml2.png', label: 'b' }
		];
		expect(dedupeBurstScreenshots(shots)).toEqual([shots[1]]);
	});

	it('prefers per-test visual evidence urls', () => {
		const screenshots = [
			{ url: '/a.png', label: 'a' },
			{ url: '/b.png', label: 'b' }
		];
		const test: TestResult = {
			test_id: 'WS_RP_DM_AddressData_001',
			evidence: [{ visual: ['/b.png'] }]
		};
		expect(screenshotsForTest(screenshots, test)).toEqual([screenshots[1]]);
	});
});

describe('groupExecutedTests', () => {
	it('groups by parsed FCAF category and subgroup', () => {
		const groups = groupExecutedTests([
			{ test_id: 'WS_RP_DM_AddressData_001', status: 'passed' },
			{ test_id: 'WS_RP_DM_AddressData_002', status: 'failed' },
			{ test_id: 'WS_RP_IA_MainInteraction__003', status: 'passed' }
		]);
		expect(groups.map((g) => g.category.code)).toEqual(['DM', 'IA']);
		expect(groups[0].groups[0]).toMatchObject({
			key: 'addressdata',
			passed: 1,
			total: 2,
			rate: 50
		});
	});
});

describe('prepareReportDisplay', () => {
	const report: Report = {
		status: 'failed',
		suite: 'ws_rp',
		summary: { passed: 1, failed: 1 },
		evidence: {
			deeplink: 'openid4vp://request',
			shot: 'https://example.com/shared.png'
		},
		executed_tests: [
			{
				test_id: 'WS_RP_DM_AddressData_Email_001',
				title: 'Email claim',
				status: 'passed',
				evidence: [{ visual: ['https://example.com/shared.png'] }]
			},
			{
				test_id: 'WS_RP_IA_MainInteraction__003',
				title: 'Main flow',
				status: 'failed'
			}
		]
	};

	it('builds filtered display data', () => {
		const display = prepareReportDisplay(report, [], {
			filter: 'fail',
			searchQuery: ''
		});
		expect(display.totalTests).toBe(2);
		expect(display.passedTests).toBe(1);
		expect(display.failedTests).toBe(1);
		expect(display.filteredTests).toHaveLength(1);
		expect(display.filteredTests[0].test_id).toBe('WS_RP_IA_MainInteraction__003');
		expect(display.checkedDeeplink).toBe('openid4vp://request');
		expect(display.summaryEntries).toEqual([
			['passed', 1],
			['failed', 1]
		]);
	});

	it('filters by search query', () => {
		expect(testMatchesSearch(report.executed_tests![0], 'email')).toBe(true);
		expect(testMatchesSearch(report.executed_tests![0], 'interaction')).toBe(false);
		const display = prepareReportDisplay(report, [], {
			filter: 'all',
			searchQuery: 'interaction'
		});
		expect(display.filteredTests).toHaveLength(1);
		expect(display.groupedCategories[0].category.code).toBe('IA');
	});

	it('collects maestro and evidence screenshots', () => {
		const shots = collectScreenshots(report, ['https://example.com/maestro.png']);
		expect(shots.map((s) => s.url)).toEqual([
			'https://example.com/maestro.png',
			'https://example.com/shared.png'
		]);
	});
});

describe('sourceUrl', () => {
	it('builds the FCAF docs anchor', () => {
		expect(sourceUrl('WS_RP_DM_AddressData_001')).toBe(
			'https://conformance.eudi.dev/latest-draft/fcaf/suts/wallet_solution/relying_party/ws_rp/#ws_rp_dm_addressdata_001'
		);
		expect(sourceUrl(undefined)).toBeUndefined();
	});

	it('finds nested deeplinks', () => {
		expect(evidenceDeeplink({ a: { deeplink: 'x://y' } })).toBe('x://y');
	});
});
