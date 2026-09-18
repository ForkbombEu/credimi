// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * @deprecated Prefer `import { FCAF } from '$lib'` (`FCAF.groupAllTests`, etc.).
 * Thin re-export for leftover relative imports inside this step package.
 */
export {
	groupAllTests,
	groupSelectedTests,
	groupTests,
	type FCAFGroupedTests,
	type FCAFSubgroupTests
} from '../../../fcaf/catalog.js';
