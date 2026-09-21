// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	groupExecutedTests,
	prepareReportDisplay,
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
	it('selects presentation screenshots assigned to the test', () => {
		const screenshots = [
			{ url: '/a.png', label: 'a', test_ids: ['other-test'] },
			{ url: '/b.png', label: 'b', test_ids: ['test-1', 'other-test'] },
			{ url: '/unassigned.png', label: 'unassigned' }
		];
		const test: TestResult = { test_id: 'test-1' };
		expect(screenshotsForTest(screenshots, test)).toEqual([{ url: '/b.png', label: 'b' }]);
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
		summary: { passed: 99, failed: 99 },
		evidence: {
			deeplink: 'legacy://ignored',
			shot: 'https://example.com/legacy-ignored.png'
		},
		presentation: {
			deeplink: 'openid4vp://request',
			screenshots: [
				{
					url: 'https://example.com/passed.png',
					label: 'Passed evidence',
					test_ids: ['WS_RP_DM_AddressData_Email_001']
				},
				{
					url: 'https://example.com/failed.png',
					label: 'Failed evidence',
					test_ids: ['WS_RP_IA_MainInteraction__003']
				},
				{
					url: 'https://example.com/unassigned.png',
					label: 'Unassigned evidence'
				},
				{
					url: 'https://example.com/also-unassigned.png',
					label: 'Also unassigned',
					test_ids: []
				}
			],
			summary_filters: [
				{ key: 'passed', label: 'Passed', count: 1 },
				{ key: 'failed', label: 'Failed', count: 1 }
			]
		},
		executed_tests: [
			{
				test_id: 'WS_RP_DM_AddressData_Email_001',
				title: 'Email claim',
				status: 'passed'
			},
			{
				test_id: 'WS_RP_IA_MainInteraction__003',
				title: 'Main flow',
				status: 'failed'
			}
		]
	};

	it('builds filtered display data', () => {
		const display = prepareReportDisplay(report, {
			filter: 'failed',
			searchQuery: ''
		});
		expect(display.totalTests).toBe(2);
		expect(display.passedTests).toBe(1);
		expect(display.failedTests).toBe(1);
		expect(display.filteredTests).toHaveLength(1);
		expect(display.filteredTests[0].test_id).toBe('WS_RP_IA_MainInteraction__003');
		expect(display.checkedDeeplink).toBe('openid4vp://request');
		expect(display.summaryFilters).toEqual([
			{ key: 'passed', label: 'Passed', count: 1 },
			{ key: 'failed', label: 'Failed', count: 1 }
		]);
		expect(display.allScreenshots).toEqual([
			{ url: 'https://example.com/passed.png', label: 'Passed evidence' },
			{ url: 'https://example.com/failed.png', label: 'Failed evidence' },
			{ url: 'https://example.com/unassigned.png', label: 'Unassigned evidence' },
			{ url: 'https://example.com/also-unassigned.png', label: 'Also unassigned' }
		]);
		expect(display.unassignedScreenshots).toEqual([
			{ url: 'https://example.com/unassigned.png', label: 'Unassigned evidence' },
			{ url: 'https://example.com/also-unassigned.png', label: 'Also unassigned' }
		]);
	});

	it('filters by search query', () => {
		expect(testMatchesSearch(report.executed_tests![0], 'email')).toBe(true);
		expect(testMatchesSearch(report.executed_tests![0], 'interaction')).toBe(false);
		const display = prepareReportDisplay(report, {
			filter: 'all',
			searchQuery: 'interaction'
		});
		expect(display.filteredTests).toHaveLength(1);
		expect(display.groupedCategories[0].category.code).toBe('IA');
	});

	it('uses exact presentation filter keys', () => {
		const display = prepareReportDisplay(
			{
				presentation: {
					summary_filters: [{ key: 'passed', label: 'Passed', count: 1 }]
				},
				executed_tests: [
					{ test_id: 'WS_RP_DM_AddressData_001', status: 'passed' },
					{ test_id: 'WS_RP_DM_AddressData_002', status: 'passed-with-warning' }
				]
			},
			{ filter: 'passed', searchQuery: '' }
		);
		expect(display.filteredTests.map((test) => test.test_id)).toEqual([
			'WS_RP_DM_AddressData_001'
		]);
	});

	it('renders historical reports when presentation is missing', () => {
		const display = prepareReportDisplay(
			{
				evidence: {
					session: {
						deeplink: 'legacy://request',
						artifacts: ['https://example.com/legacy-only.png']
					}
				},
				executed_tests: [
					{
						test_id: 'WS_RP_DM_AddressData_001',
						status: 'passed',
						assertions: [{ evidence_keys: ['session'] }]
					}
				]
			},
			{ filter: 'all', searchQuery: '' }
		);
		expect(display.allScreenshots).toEqual([
			{ url: 'https://example.com/legacy-only.png', label: 'legacy only' }
		]);
		expect(display.unassignedScreenshots).toEqual([]);
		expect(display.checkedDeeplink).toBe('legacy://request');
		expect(display.summaryFilters).toEqual([{ key: 'passed', label: 'Passed', count: 1 }]);
		expect(display.executedTests).toHaveLength(1);
	});

	it('fills missing presentation fields from historical report data', () => {
		const display = prepareReportDisplay(
			{
				evidence: {
					visual_evidence: {
						artifacts: ['https://example.com/fallback.png?token=temporary']
					}
				},
				presentation: {
					deeplink: 'openid4vp://request'
				},
				executed_tests: [
					{
						test_id: 'WS_RP_DM_AddressData_001',
						status: 'passed',
						assertions: [{ evidence_keys: ['visual_evidence'] }]
					}
				]
			},
			{ filter: 'all', searchQuery: '' }
		);
		expect(display.checkedDeeplink).toBe('openid4vp://request');
		expect(display.summaryFilters).toEqual([{ key: 'passed', label: 'Passed', count: 1 }]);
		expect(display.allScreenshots).toEqual([
			{ url: 'https://example.com/fallback.png', label: 'fallback' }
		]);
		expect(display.unassignedScreenshots).toEqual([]);
	});
});

describe('sourceUrl', () => {
	it('builds the FCAF docs anchor', () => {
		expect(sourceUrl('WS_RP_DM_AddressData_001')).toBe(
			'https://conformance.eudi.dev/latest-draft/fcaf/suts/wallet_solution/relying_party/ws_rp/#ws_rp_dm_addressdata_001'
		);
		expect(sourceUrl(undefined)).toBeUndefined();
	});
});
