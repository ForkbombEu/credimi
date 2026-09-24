// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	awaitTask,
	listChecks,
	type ListChecksError,
	type ListChecksOptions
} from '$lib/conformance/client.js';
import type { ConformanceCheckRecord } from '$lib/conformance/record.js';
import * as Task from 'true-myth/task';

import { FCAF_CATEGORY_ORDER, parseFCAFTestId, type FCAFCategory } from './categories.js';

/**
 * Catalog entry for FCAF listing/picking (from conformance_checks).
 * Grouping uses the test id prefix (FCAF taxonomy); section/sources are not on this shape.
 */
export type FCAFTestCatalogEntry = {
	id: string;
	title: string;
};

/** Standard uid for FCAF rows in `conformance_checks`. */
export const FCAF_STANDARD = 'fcaf' as const;

/**
 * FCAF test id is the final segment of the catalog path
 * (`fcaf/<version>/<suite>/<test_id>`).
 */
export function fcafTestIdFromPath(path: string): string {
	const segments = path.split('/').filter(Boolean);
	return segments[segments.length - 1] ?? path;
}

/** Map a catalog row to the FCAF listing/picker entry shape. */
export function toFcafCatalogEntry(record: ConformanceCheckRecord): FCAFTestCatalogEntry {
	return {
		id: fcafTestIdFromPath(record.path),
		title: record.title
	};
}

export type ListFcafTestsOptions = Omit<ListChecksOptions, 'fs_standard'>;

/** List FCAF tests from the shared conformance catalog (not static codegen). */
export function listFcafTests(
	options: ListFcafTestsOptions
): Task.Task<FCAFTestCatalogEntry[], ListChecksError> {
	return listChecks({ ...options, fs_standard: FCAF_STANDARD }).map((records) =>
		records.map(toFcafCatalogEntry)
	);
}

/** Promise helper for loaders / TanStack queryFn. */
export async function getFcafTests(
	options: ListFcafTestsOptions
): Promise<FCAFTestCatalogEntry[] | Error> {
	return awaitTask(listFcafTests(options));
}

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

export function groupSelectedTests(testIds: string[]): CatalogCategoryGroup[] {
	// Card summaries only have selected ids; synthesize minimal entries for grouping.
	return groupCatalogTests(
		testIds.map((id) => ({ id, title: '' }))
	);
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
