// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Public FCAF surface. Prefer `import { FCAF } from '$lib'` from routes and
 * cross-module callers; use relative imports inside this folder.
 */

export {
	FCAF_CATEGORY_ORDER as CATEGORY_ORDER,
	parseFCAFTestId as parseTestId,
	subgroupLabel,
	type FCAFCategory as Category,
	type FCAFCategoryCode as CategoryCode,
	type ParsedFCAFTestId as ParsedTestId
} from './categories.js';

export {
	groupAllTests,
	groupSelectedTests,
	groupCatalogTests,
	groupTests,
	type CatalogCategoryGroup,
	type CatalogSubgroup,
	type FCAFGroupedTests,
	type FCAFSubgroupTests
} from './catalog.js';

export {
	clearReportCache,
	collectScreenshots,
	dedupeBurstScreenshots,
	evidenceDeeplink,
	evidenceScreenshotUrls,
	groupExecutedTests,
	loadReport,
	prepareReportDisplay,
	screenshotLabel,
	screenshotsForTest,
	screenshotsWithoutTest,
	sourceUrl,
	statusClass,
	statusDotClass,
	statusIsFailed,
	statusIsPassed,
	testMatchesSearch,
	uniqueScreenshots,
	type CategoryGroup,
	type ExecutedEvidence,
	type GroupedGroup,
	type Report,
	type ReportDisplay,
	type Screenshot,
	type TestResult,
	type Validator
} from './report.js';

export {
	FCAF_PIPELINE_OUTPUTS as PIPELINE_OUTPUTS,
	FCAF_SUITE as SUITE,
	FCAF_TESTS as TESTS,
	type FCAFTestCatalogEntry as TestCatalogEntry
} from './tests.generated.js';

import ReportSheet from './report-sheet.svelte';
import ReportView from './report-view.svelte';

export { ReportSheet, ReportView };
