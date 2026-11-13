// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { FCAF_CATEGORY_ORDER, parseFCAFTestId, type FCAFCategory } from './categories.js';

export type TestResult = {
	test_id?: string;
	title?: string;
	/** FCAF source test link, set by the backend from the source heading */
	source_url?: string;
	status?: string;
	assertions?: Array<{
		id?: string;
		status?: string;
		message?: string;
		validator?: string;
		evidence_keys?: string[];
	}>;
	/** `key` is the record's entry in `Report.evidence`; reports stored before it existed key that map by `name`. */
	evidence?: Array<{ name?: string; key?: string; visual?: string[] }>;
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
	displayedCount: number;
	displayedLabel: string;
	rate: number;
	tests: TestResult[];
};

export type CategoryGroup = {
	category: FCAFCategory;
	passed: number;
	total: number;
	displayedCount: number;
	displayedLabel: string;
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

type LegacyScreenshot = PresentationScreenshot;

const reportCache = new Map<string, Promise<Report | undefined>>();

const RESULT_FILES_PATH = '/api/files/pipeline_results/';

export function loadReport(url: string): Promise<Report | undefined> {
	let cached = reportCache.get(url);
	if (!cached) {
		cached = fetch(url)
			.then(async (response) => {
				if (!response.ok) throw new Error(`FCAF report request failed: ${response.status}`);
				return rebaseResultFileURLs((await response.json()) as Report, url);
			})
			.catch(() => undefined);
		reportCache.set(url, cached);
	}
	return cached;
}

/**
 * A stored report embeds absolute pipeline result file URLs from the moment it was generated;
 * some carry an origin browsers cannot reach (e.g. localhost). The report itself is served from
 * the current public base URL, so every result file URL is rebased onto that base. This mirrors
 * the PDF renderer, which resolves screenshots by filename on the pipeline result record.
 */
export function rebaseResultFileURLs<T>(value: T, reportUrl: string): T {
	const baseEnd = reportUrl.indexOf(RESULT_FILES_PATH);
	if (baseEnd < 0) return value;
	return rebaseResultFileValue(value, reportUrl.slice(0, baseEnd)) as T;
}

function rebaseResultFileValue(value: unknown, base: string): unknown {
	if (typeof value === 'string') {
		const pathStart = value.indexOf(RESULT_FILES_PATH);
		if (pathStart < 0 || !/^https?:\/\//i.test(value)) return value;
		return base + value.slice(pathStart);
	}
	if (Array.isArray(value)) return value.map((child) => rebaseResultFileValue(child, base));
	if (value && typeof value === 'object') {
		return Object.fromEntries(
			Object.entries(value).map(([key, child]) => [key, rebaseResultFileValue(child, base)])
		);
	}
	return value;
}

export type TestCheck = NonNullable<TestResult['assertions']>[number];

/** Prefer non-empty assertions; fall back to historical validators. */
export function checksForTest(test: TestResult): TestCheck[] {
	if ((test.assertions?.length ?? 0) > 0) {
		return test.assertions ?? [];
	}
	return test.validators ?? [];
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

function appendToTestGroup(groups: Map<string, TestResult[]>, key: string, test: TestResult): void {
	const existing = groups.get(key);
	if (existing) {
		existing.push(test);
		return;
	}
	groups.set(key, [test]);
}

export function groupExecutedTests(
	tests: TestResult[],
	options: {
		countedTests?: TestResult[];
		visibleTests?: TestResult[];
		displayedLabel?: string;
	} = {}
): CategoryGroup[] {
	const countedTests =
		options.countedTests ?? tests.filter((test) => statusIsPassed(test.status));
	const visibleTests = options.visibleTests ?? tests;
	const displayedLabel = options.displayedLabel ?? 'passed';
	const testsByGroup = new Map<string, TestResult[]>();
	const visibleTestsByGroup = new Map<string, TestResult[]>();
	const groupKey = (test: TestResult) => {
		const parsed = parseFCAFTestId(test.test_id);
		return `${parsed.category.code}/${parsed.key}`;
	};
	for (const test of countedTests) {
		appendToTestGroup(testsByGroup, groupKey(test), test);
	}
	for (const test of visibleTests) {
		appendToTestGroup(visibleTestsByGroup, groupKey(test), test);
	}
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
	return FCAF_CATEGORY_ORDER.flatMap((code) => {
		const entry = byCategory.get(code);
		if (!entry) return [];
		const groups: GroupedGroup[] = [...entry.groups.entries()]
			.sort(([a], [b]) => a.localeCompare(b))
			.flatMap(([key, { label, tests: groupTests }]) => {
				const groupKeyValue = `${entry.category.code}/${key}`;
				const matchingTests = testsByGroup.get(groupKeyValue) ?? [];
				const groupVisibleTests = visibleTestsByGroup.get(groupKeyValue) ?? [];
				if (groupVisibleTests.length === 0) return [];
				const passed = groupTests.filter((t) => statusIsPassed(t.status)).length;
				const total = groupTests.length;
				return [
					{
						key,
						label,
						passed,
						total,
						displayedCount: matchingTests.length,
						displayedLabel,
						rate: total > 0 ? Math.round((matchingTests.length / total) * 100) : 0,
						tests: groupVisibleTests
					}
				];
			});
		if (groups.length === 0) return [];
		return [
			{
				category: entry.category,
				passed: groups.reduce((sum, group) => sum + group.passed, 0),
				total: groups.reduce((sum, group) => sum + group.total, 0),
				displayedCount: groups.reduce((sum, group) => sum + group.displayedCount, 0),
				displayedLabel,
				groups
			}
		];
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
	const presentationScreenshots = mergePresentationScreenshots(
		report.presentation?.screenshots ?? [],
		legacyPresentationScreenshots(report)
	);
	const allScreenshots = presentationScreenshots.map(({ url, label }) => ({ url, label }));
	const executedTests = report.executed_tests ?? [];
	const totalTests = executedTests.length;
	const passedTests = executedTests.filter((t) => statusIsPassed(t.status)).length;
	const failedTests = executedTests.filter((t) => statusIsFailed(t.status)).length;
	const otherTests = totalTests - passedTests - failedTests;
	const statusFilteredTests = executedTests.filter(
		(t) => options.filter === 'all' || t.status === options.filter
	);
	const countedTests =
		options.filter === 'all'
			? executedTests.filter((t) => statusIsPassed(t.status))
			: statusFilteredTests;
	const filteredTests = statusFilteredTests.filter((t) =>
		testMatchesSearch(t, options.searchQuery)
	);
	const groupedCategories = groupExecutedTests(executedTests, {
		countedTests,
		visibleTests: filteredTests,
		displayedLabel: options.filter === 'all' ? 'passed' : options.filter.replaceAll('_', ' ')
	});
	const unassignedScreenshots = presentationScreenshots
		.filter(({ test_ids }) => !test_ids?.length)
		.map(({ url, label }) => ({ url, label }));
	const checkedDeeplink = report.presentation?.deeplink ?? legacyDeeplink(report.evidence);
	const summaryFilters = report.presentation?.summary_filters?.length
		? report.presentation.summary_filters
		: legacySummaryFilters(report);

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

function mergePresentationScreenshots(
	presentation: PresentationScreenshot[],
	legacy: LegacyScreenshot[]
): PresentationScreenshot[] {
	const merged = new Map<string, PresentationScreenshot>();
	for (const screenshot of legacy) {
		merged.set(screenshot.url, {
			url: screenshot.url,
			label: screenshot.label,
			test_ids: [...(screenshot.test_ids ?? [])]
		});
	}
	for (const screenshot of presentation) {
		const existing = merged.get(screenshot.url);
		if (!existing) {
			merged.set(screenshot.url, {
				url: screenshot.url,
				label: screenshot.label,
				test_ids: [...(screenshot.test_ids ?? [])]
			});
			continue;
		}
		existing.label = screenshot.label || existing.label;
		existing.test_ids = [
			...new Set([...(existing.test_ids ?? []), ...(screenshot.test_ids ?? [])])
		];
	}
	return [...merged.values()];
}

function legacyPresentationScreenshots(report: Report): LegacyScreenshot[] {
	const screenshots = new Map<string, LegacyScreenshot>();
	const assignments = legacyScreenshotAssignments(report);
	for (const [key, value] of Object.entries(report.evidence ?? {})) {
		for (const screenshot of imageReferences(value, assignments.get(key) ?? [])) {
			const existing = screenshots.get(screenshot.url);
			if (!existing) {
				screenshots.set(screenshot.url, screenshot);
				continue;
			}
			existing.test_ids = [
				...new Set([...(existing.test_ids ?? []), ...(screenshot.test_ids ?? [])])
			];
		}
	}
	for (const test of report.executed_tests ?? []) {
		for (const item of test.evidence ?? []) {
			for (const reference of item.visual ?? []) {
				const normalized = normalizePresentationURL(reference);
				if (!normalized) continue;
				const existing = screenshots.get(normalized);
				const testIds = test.test_id ? [test.test_id] : [];
				if (!existing) {
					screenshots.set(normalized, {
						url: normalized,
						label: presentationLabel(normalized),
						test_ids: testIds
					});
					continue;
				}
				existing.test_ids = [...new Set([...(existing.test_ids ?? []), ...testIds])];
			}
		}
	}
	return [...screenshots.values()];
}

function legacyScreenshotAssignments(report: Report): Map<string, string[]> {
	const assignments = new Map<string, string[]>();
	for (const test of report.executed_tests ?? []) {
		const testId = test.test_id;
		if (!testId) continue;
		const keys = new Set<string>();
		const recordKeys = new Map<string, string>();
		// Reports stored before evidence records carried a key use evidence names as map keys.
		const keyedByName = !(test.evidence ?? []).some((item) => item.key);
		for (const item of test.evidence ?? []) {
			const key = item.key ?? item.name;
			if (item.name && key) recordKeys.set(item.name, key);
			if (key) keys.add(key);
		}
		for (const assertion of test.assertions ?? []) {
			for (const name of assertion.evidence_keys ?? []) {
				const key = recordKeys.get(name) ?? (keyedByName ? name : undefined);
				if (key) keys.add(key);
			}
		}
		for (const key of keys) {
			assignments.set(key, [...new Set([...(assignments.get(key) ?? []), testId])]);
		}
	}
	return assignments;
}

function imageReferences(value: unknown, testIds: string[]): LegacyScreenshot[] {
	if (typeof value === 'string') {
		if (!isPresentationImage(value)) return [];
		const normalized = normalizePresentationURL(value);
		if (!normalized) return [];
		return [{ url: normalized, label: presentationLabel(normalized), test_ids: [...testIds] }];
	}
	if (Array.isArray(value)) {
		const nested = value.flatMap((child) => imageReferences(child, testIds));
		return dedupeLegacyScreenshots(nested);
	}
	if (value && typeof value === 'object') {
		const nested = Object.values(value).flatMap((child) => imageReferences(child, testIds));
		return dedupeLegacyScreenshots(nested);
	}
	return [];
}

function dedupeLegacyScreenshots(screenshots: LegacyScreenshot[]): LegacyScreenshot[] {
	const deduped = new Map<string, LegacyScreenshot>();
	for (const screenshot of screenshots) {
		const existing = deduped.get(screenshot.url);
		if (!existing) {
			deduped.set(screenshot.url, screenshot);
			continue;
		}
		existing.test_ids = [
			...new Set([...(existing.test_ids ?? []), ...(screenshot.test_ids ?? [])])
		];
	}
	return [...deduped.values()];
}

function isPresentationImage(reference: string): boolean {
	try {
		const parsed = new URL(reference, 'https://credimi.invalid');
		return ['.jpeg', '.jpg', '.png', '.webp'].includes(
			parsed.pathname.slice(parsed.pathname.lastIndexOf('.')).toLowerCase()
		);
	} catch {
		return false;
	}
}

function normalizePresentationURL(reference: string): string | undefined {
	const trimmed = reference.trim();
	if (!trimmed) return undefined;
	return trimmed.split('?')[0];
}

function presentationLabel(reference: string): string {
	const filename = decodeURIComponent(reference.split('/').pop() ?? '')
		.replace(/\.[^.]+$/, '')
		.replaceAll('_', ' ')
		.replaceAll('-', ' ')
		.trim();
	return filename.split(/\s+/).filter(Boolean).join(' ');
}

function legacyDeeplink(evidence: Record<string, unknown> | undefined): string | undefined {
	if (!evidence) return undefined;
	for (const key of Object.keys(evidence).sort()) {
		const deeplink = nestedDeeplink(evidence[key]);
		if (deeplink) return deeplink;
	}
	return undefined;
}

function nestedDeeplink(value: unknown): string | undefined {
	if (Array.isArray(value)) {
		for (const nested of value) {
			const deeplink = nestedDeeplink(nested);
			if (deeplink) return deeplink;
		}
		return undefined;
	}
	if (value && typeof value === 'object') {
		for (const [key, nested] of Object.entries(value).sort(([left], [right]) =>
			left.localeCompare(right)
		)) {
			if (key === 'deeplink' && typeof nested === 'string') return nested;
			const deeplink = nestedDeeplink(nested);
			if (deeplink) return deeplink;
		}
	}
	return undefined;
}

function legacySummaryFilters(
	report: Report
): Array<{ key: string; label: string; count: number }> {
	const counts = new Map<string, number>();
	for (const test of report.executed_tests ?? []) {
		const key = test.status;
		if (!key) continue;
		counts.set(key, (counts.get(key) ?? 0) + 1);
	}
	return [
		{ key: 'passed', label: 'Passed' },
		{ key: 'failed', label: 'Failed' },
		{ key: 'blocked', label: 'Blocked' },
		{ key: 'skipped', label: 'Skipped' },
		{ key: 'inconclusive', label: 'Inconclusive' }
	]
		.map((filter) => ({ ...filter, count: counts.get(filter.key) ?? 0 }))
		.filter((filter) => filter.count > 0);
}
