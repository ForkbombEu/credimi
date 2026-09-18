// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Public FCAF surface. Prefer `import { FCAF } from '$lib'` from routes and
 * cross-module callers; use relative imports inside this folder.
 *
 * Keep this barrel small: only what outside callers need. Report helpers and
 * ReportView stay private to `$lib/fcaf`.
 */

export {
	groupAllTests,
	groupSelectedTests,
	groupCatalogTests,
	type CatalogCategoryGroup,
	type CatalogSubgroup
} from './catalog.js';

export {
	FCAF_PIPELINE_OUTPUTS as PIPELINE_OUTPUTS,
	FCAF_SUITE as SUITE,
	FCAF_TESTS as TESTS,
	type FCAFTestCatalogEntry as TestCatalogEntry
} from './tests.generated.js';

import ReportSheet from './report-sheet.svelte';

export { ReportSheet };
