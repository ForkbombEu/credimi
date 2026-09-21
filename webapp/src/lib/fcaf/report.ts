// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { FCAF_CATEGORY_ORDER, parseFCAFTestId, type FCAFCategory } from './categories.js';

export type TestResult = {
	test_id?: string;
	title?: string;
	status?: string;
	assertions?: Array<{ id?: string; status?: string; message?: string; validator?: string }>;
	/** @deprecated historical reports only; prefer assertions */
	validators?: Array<{ id?: string; status?: string; message?: string; validator?: string }>;
};

export type PresentationScreenshot = {
	url: string;
	label: string;
	test_ids?: string[];
};

export type Report = {
	status?: string;
	suite?: string;
	summary?: Record<string, number>;
	evidence?: Record<string, unknown>;
	executed_tests?: TestResult[];
	presentation?: {
		deeplink?: string;
		screenshots?: PresentationScreenshot[];
		summary_filters?: Array<{ key: string; label: string; count: number }>;
	};
};

export type Screenshot = {
	url: string;
	label: string;
};

export type GroupedGroup = {
	key: string;
	label: string;
	passed: number;
	total: number;
	rate: number;
	tests: TestResult[];
};

export type CategoryGroup = {
	category: FCAFCategory;
	passed: number;
	total: number;
	groups: GroupedGroup[];
};

export type ReportDisplay = {
	allScreenshots: Screenshot[];
	executedTests: TestResult[];
	totalTests: number;
	passedTests: number;
	failedTests: number;
	otherTests: number;
	filteredTests: TestResult[];
	groupedCategories: CategoryGroup[];
	unassignedScreenshots: Screenshot[];
	checkedDeeplink: string | undefined;
	summaryFilters: Array<{ key: string; label: string; count: number }>;
};

const reportCache = new Map<string, Promise<Report | undefined>>();

export function loadReport(url: string): Promise<Report | undefined> {
	let cached = reportCache.get(url);
	if (!cached) {
		cached = fetch(url)
			.then(async (response) => {
				if (!response.ok) throw new Error(`FCAF report request failed: ${response.status}`);
				return (await response.json()) as Report;
			})
			.catch(() => undefined);
		reportCache.set(url, cached);
	}
	return cached;
}

export function statusIsPassed(status = ''): boolean {
	return (status ?? '').startsWith('pass');
}

export function statusIsFailed(status = ''): boolean {
	return ['failed', 'fail', 'error'].includes(status ?? '');
}

export function statusClass(status = ''): string {
	if (['passed', 'pass'].includes(status)) return 'text-green-700';
	if (['failed', 'fail', 'error'].includes(status)) return 'text-red-700';
	return 'text-amber-700';
}

export function statusDotClass(status = ''): string {
	if (['passed', 'pass'].includes(status)) return 'bg-green-600';
	if (['failed', 'fail', 'error'].includes(status)) return 'bg-red-600';
	return 'bg-amber-500';
}

export function screenshotsForTest(
	screenshots: PresentationScreenshot[],
	test: TestResult
): Screenshot[] {
	const testId = test.test_id;
	if (!testId) return [];
	return screenshots
		.filter(({ test_ids }) => test_ids?.includes(testId) ?? false)
		.map(({ url, label }) => ({ url, label }));
}

export function sourceUrl(testId: string | undefined): string | undefined {
	if (!testId) return undefined;
	const anchor = testId
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '_')
		.replace(/^_+|_+$/g, '');
	return `https://conformance.eudi.dev/latest-draft/fcaf/suts/wallet_solution/relying_party/ws_rp/#${anchor}`;
}

export function groupExecutedTests(tests: TestResult[]): CategoryGroup[] {
	const byCategory = new Map<
		string,
		{ category: FCAFCategory; groups: Map<string, { label: string; tests: TestResult[] }> }
	>();
	for (const test of tests) {
		const parsed = parseFCAFTestId(test.test_id);
		let entry = byCategory.get(parsed.category.code);
		if (!entry) {
			entry = { category: parsed.category, groups: new Map() };
			byCategory.set(parsed.category.code, entry);
		}
		const bucket = entry.groups.get(parsed.key);
		if (bucket) {
			bucket.tests.push(test);
		} else {
			entry.groups.set(parsed.key, { label: parsed.label, tests: [test] });
		}
	}
	return FCAF_CATEGORY_ORDER.filter((code) => byCategory.has(code)).map((code) => {
		const entry = byCategory.get(code)!;
		const groups: GroupedGroup[] = [...entry.groups.entries()]
			.sort(([a], [b]) => a.localeCompare(b))
			.map(([key, { label, tests: groupTests }]) => {
				const passed = groupTests.filter((t) => statusIsPassed(t.status)).length;
				const total = groupTests.length;
				return {
					key,
					label,
					passed,
					total,
					rate: total > 0 ? Math.round((passed / total) * 100) : 0,
					tests: groupTests
				};
			});
		return {
			category: entry.category,
			passed: groups.reduce((sum, group) => sum + group.passed, 0),
			total: groups.reduce((sum, group) => sum + group.total, 0),
			groups
		};
	});
}

export function testMatchesSearch(test: TestResult, query: string): boolean {
	if (!query) return true;
	const parsed = parseFCAFTestId(test.test_id);
	const haystack =
		`${test.test_id ?? ''} ${test.title ?? ''} ${parsed.category.label} ${parsed.label}`.toLowerCase();
	return haystack.includes(query);
}

export function prepareReportDisplay(
	report: Report,
	options: { filter: string; searchQuery: string }
): ReportDisplay {
	const presentationScreenshots = report.presentation?.screenshots ?? [];
	const allScreenshots = presentationScreenshots.map(({ url, label }) => ({ url, label }));
	const executedTests = report.executed_tests ?? [];
	const totalTests = executedTests.length;
	const passedTests = executedTests.filter((t) => statusIsPassed(t.status)).length;
	const failedTests = executedTests.filter((t) => statusIsFailed(t.status)).length;
	const otherTests = totalTests - passedTests - failedTests;
	const filteredTests = executedTests.filter(
		(t) =>
			(options.filter === 'all' || t.status === options.filter) &&
			testMatchesSearch(t, options.searchQuery)
	);
	const groupedCategories = groupExecutedTests(filteredTests);
	const unassignedScreenshots = presentationScreenshots
		.filter(({ test_ids }) => !test_ids?.length)
		.map(({ url, label }) => ({ url, label }));
	const checkedDeeplink = report.presentation?.deeplink;
	const summaryFilters = report.presentation?.summary_filters ?? [];

	return {
		allScreenshots,
		executedTests,
		totalTests,
		passedTests,
		failedTests,
		otherTests,
		filteredTests,
		groupedCategories,
		unassignedScreenshots,
		checkedDeeplink,
		summaryFilters
	};
}
