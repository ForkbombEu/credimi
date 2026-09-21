// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { FCAF_CATEGORY_ORDER, parseFCAFTestId, type FCAFCategory } from './categories.js';

export type TestResult = {
	test_id?: string;
	title?: string;
	status?: string;
	assertions?: Array<{
		id?: string;
		status?: string;
		message?: string;
		validator?: string;
		evidence_keys?: string[];
	}>;
	evidence?: Array<{ name?: string; visual?: string[] }>;
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

type LegacyScreenshot = PresentationScreenshot;

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
	const filteredTests = executedTests.filter(
		(t) =>
			(options.filter === 'all' || t.status === options.filter) &&
			testMatchesSearch(t, options.searchQuery)
	);
	const groupedCategories = groupExecutedTests(filteredTests);
	const unassignedScreenshots = presentationScreenshots
		.filter(({ test_ids }) => !test_ids?.length)
		.map(({ url, label }) => ({ url, label }));
	const checkedDeeplink = report.presentation?.deeplink ?? legacyDeeplink(report.evidence);
	const summaryFilters =
		report.presentation?.summary_filters?.length ? report.presentation.summary_filters : legacySummaryFilters(report);

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
		existing.test_ids = [...new Set([...(existing.test_ids ?? []), ...(screenshot.test_ids ?? [])])];
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
			existing.test_ids = [...new Set([...(existing.test_ids ?? []), ...(screenshot.test_ids ?? [])])];
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
		for (const assertion of test.assertions ?? []) {
			for (const key of assertion.evidence_keys ?? []) {
				if (key) keys.add(key);
			}
		}
		for (const item of test.evidence ?? []) {
			if (item.name) keys.add(item.name);
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
		existing.test_ids = [...new Set([...(existing.test_ids ?? []), ...(screenshot.test_ids ?? [])])];
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

function legacySummaryFilters(report: Report): Array<{ key: string; label: string; count: number }> {
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
