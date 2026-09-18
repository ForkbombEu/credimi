// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { FCAF_CATEGORY_ORDER, parseFCAFTestId, type FCAFCategory } from './categories.js';

export type Validator = {
	id?: string;
	status?: string;
	message?: string;
	validator?: string;
};

export type ExecutedEvidence = {
	name?: string;
	source_node?: string;
	visual?: string[];
};

export type TestResult = {
	test_id?: string;
	title?: string;
	status?: string;
	assertions?: Array<{ id?: string; status?: string; message?: string; validator?: string }>;
	validators?: Validator[];
	evidence?: ExecutedEvidence[];
};

export type Report = {
	status?: string;
	suite?: string;
	summary?: Record<string, number>;
	evidence?: Record<string, unknown>;
	executed_tests?: TestResult[];
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
	summaryEntries: Array<[string, number]>;
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

/** Test-only: clear the in-memory report fetch cache. */
export function clearReportCache() {
	reportCache.clear();
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

export function evidenceScreenshotUrls(value: unknown): string[] {
	const urls: string[] = [];
	const visit = (current: unknown) => {
		if (typeof current === 'string' && /\.(?:png|jpe?g|webp)(?:\?|$)/i.test(current)) {
			urls.push(current);
			return;
		}
		if (Array.isArray(current)) current.forEach(visit);
		else if (current && typeof current === 'object') {
			Object.values(current as Record<string, unknown>).forEach(visit);
		}
	};
	visit(value);
	return [...new Set(urls)];
}

export function screenshotLabel(url: string): string {
	const filename = decodeURIComponent(url.split('?')[0].split('/').pop() ?? url);
	return filename
		.replace(/\.[^.]+$/, '')
		.replace(/[_-]+/g, ' ')
		.replace(/\s+/g, ' ')
		.trim();
}

export function uniqueScreenshots(screenshots: Screenshot[]): Screenshot[] {
	const seen = new Set<string>();
	return screenshots.filter((screenshot) => {
		const key = screenshot.label.toLowerCase();
		if (seen.has(key)) return false;
		seen.add(key);
		return true;
	});
}

export function dedupeBurstScreenshots(screenshots: Screenshot[]): Screenshot[] {
	const burst =
		/^(.+)(?:_screenshot_\d+_action_[A-Za-z0-9_]+\.yaml\d+|_step_\d+_[A-Za-z0-9_]+)\.png$/;
	const lastOfBurst = new Map<string, Screenshot>();
	for (const screenshot of screenshots) {
		const filename = screenshot.url.split('?')[0].split('/').pop() ?? '';
		const match = burst.exec(decodeURIComponent(filename));
		if (match) lastOfBurst.set(match[1], screenshot);
	}
	return screenshots.filter((screenshot) => {
		const filename = screenshot.url.split('?')[0].split('/').pop() ?? '';
		const match = burst.exec(decodeURIComponent(filename));
		if (!match) return true;
		return lastOfBurst.get(match[1]) === screenshot;
	});
}

function visualUrlsForTest(test: TestResult): string[] {
	const urls: string[] = [];
	const seen = new Set<string>();
	for (const item of test.evidence ?? []) {
		for (const url of item.visual ?? []) {
			const normalized = url.split('?')[0];
			if (!seen.has(normalized)) {
				seen.add(normalized);
				urls.push(normalized);
			}
		}
	}
	return urls;
}

export function screenshotsForTest(screenshots: Screenshot[], test: TestResult): Screenshot[] {
	if (screenshots.length === 0) return [];

	// Prefer the per-test visual evidence recorded by the report engine:
	// the flat evidence map keeps only one record per evidence name.
	const visual = visualUrlsForTest(test);
	if (visual.length > 0) {
		const byUrl = new Map(screenshots.map((s) => [s.url.split('?')[0], s]));
		return visual
			.map((url) => byUrl.get(url))
			.filter((screenshot): screenshot is Screenshot => Boolean(screenshot));
	}
	if (screenshots.length === 1) return screenshots;

	const searchable = `${test.test_id ?? ''} ${test.title ?? ''}`.toLowerCase();
	const matching = screenshots.filter(({ label }) => {
		const words = label
			.toLowerCase()
			.split(/\s+/)
			.filter((word) => word.length > 3);
		return words.some((word) => searchable.includes(word));
	});
	return matching;
}

export function screenshotsWithoutTest(
	screenshots: Screenshot[],
	tests: TestResult[]
): Screenshot[] {
	if (tests.length <= 1) return [];
	const assigned = new Set(
		tests.flatMap((test) => screenshotsForTest(screenshots, test).map(({ url }) => url))
	);
	return screenshots.filter(({ url }) => !assigned.has(url));
}

export function sourceUrl(testId: string | undefined): string | undefined {
	if (!testId) return undefined;
	const anchor = testId
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '_')
		.replace(/^_+|_+$/g, '');
	return `https://conformance.eudi.dev/latest-draft/fcaf/suts/wallet_solution/relying_party/ws_rp/#${anchor}`;
}

export function evidenceDeeplink(value: unknown): string | undefined {
	if (value && typeof value === 'object') {
		for (const [key, nested] of Object.entries(value as Record<string, unknown>)) {
			if (key === 'deeplink' && typeof nested === 'string') return nested;
			const found = evidenceDeeplink(nested);
			if (found) return found;
		}
	}
	if (Array.isArray(value)) {
		for (const nested of value) {
			const found = evidenceDeeplink(nested);
			if (found) return found;
		}
	}
	return undefined;
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

export function collectScreenshots(report: Report, maestroScreenshotUrls: string[]): Screenshot[] {
	const evidenceScreenshots = evidenceScreenshotUrls(report.evidence);
	return dedupeBurstScreenshots(
		uniqueScreenshots(
			[...new Set([...maestroScreenshotUrls, ...evidenceScreenshots])].map((url) => ({
				url,
				label: screenshotLabel(url)
			}))
		)
	);
}

export function prepareReportDisplay(
	report: Report,
	maestroScreenshotUrls: string[],
	options: { filter: string; searchQuery: string }
): ReportDisplay {
	const allScreenshots = collectScreenshots(report, maestroScreenshotUrls);
	const executedTests = report.executed_tests ?? [];
	const totalTests = executedTests.length;
	const passedTests = executedTests.filter((t) => statusIsPassed(t.status)).length;
	const failedTests = executedTests.filter((t) => statusIsFailed(t.status)).length;
	const otherTests = totalTests - passedTests - failedTests;
	const filteredTests = executedTests.filter(
		(t) =>
			(options.filter === 'all' || (t.status ?? '').startsWith(options.filter)) &&
			testMatchesSearch(t, options.searchQuery)
	);
	const groupedCategories = groupExecutedTests(filteredTests);
	const unassignedScreenshots = screenshotsWithoutTest(allScreenshots, executedTests);
	const checkedDeeplink = evidenceDeeplink(report.evidence);
	const summaryEntries = Object.entries(report.summary ?? {}).filter(([, count]) => count > 0);

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
		summaryEntries
	};
}
