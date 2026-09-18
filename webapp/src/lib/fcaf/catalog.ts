// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { FCAF_CATEGORY_ORDER, parseFCAFTestId, type FCAFCategory } from './categories.js';
import { FCAF_TESTS, type FCAFTestCatalogEntry } from './tests.generated.js';

/**
 * Tests grouped under one FCAF category, mirroring the generated report:
 * category (Data model, Interaction, ...) > subgroup (Address data, ...).
 */
export type CatalogSubgroup = {
	key: string;
	label: string;
	tests: FCAFTestCatalogEntry[];
};

export type CatalogCategoryGroup = {
	/** Category code (DM, MS, IA, ...). */
	key: string;
	label: string;
	color: FCAFCategory;
	groups: CatalogSubgroup[];
	/** Flat test list for the whole category, for counts and filtering. */
	tests: FCAFTestCatalogEntry[];
};

/** @deprecated Use CatalogSubgroup */
export type FCAFSubgroupTests = CatalogSubgroup;
/** @deprecated Use CatalogCategoryGroup */
export type FCAFGroupedTests = CatalogCategoryGroup;

export function groupAllTests(): CatalogCategoryGroup[] {
	return groupCatalogTests(FCAF_TESTS);
}

export function groupSelectedTests(testIds: string[]): CatalogCategoryGroup[] {
	const selected = new Set(testIds);
	return groupCatalogTests(FCAF_TESTS.filter((test) => selected.has(test.id)));
}

export function groupCatalogTests(tests: FCAFTestCatalogEntry[]): CatalogCategoryGroup[] {
	const byCategory = new Map<
		string,
		{ color: FCAFCategory; groups: Map<string, CatalogSubgroup> }
	>();
	for (const test of tests) {
		const parsed = parseFCAFTestId(test.id);
		let entry = byCategory.get(parsed.category.code);
		if (!entry) {
			entry = { color: parsed.category, groups: new Map() };
			byCategory.set(parsed.category.code, entry);
		}
		let bucket = entry.groups.get(parsed.key);
		if (!bucket) {
			bucket = { key: parsed.key, label: parsed.label, tests: [] };
			entry.groups.set(parsed.key, bucket);
		}
		bucket.tests.push(test);
	}

	return FCAF_CATEGORY_ORDER.filter((code) => byCategory.has(code)).map((code) => {
		const entry = byCategory.get(code)!;
		const groups = [...entry.groups.values()].sort((a, b) => a.key.localeCompare(b.key));
		return {
			key: code,
			label: entry.color.label,
			color: entry.color,
			groups,
			tests: groups.flatMap((group) => group.tests)
		};
	});
}

/** @deprecated Use groupCatalogTests */
export const groupTests = groupCatalogTests;
