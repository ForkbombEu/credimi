// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { FCAFTestCatalogEntry } from '$lib/fcaf/catalog.js';

import * as Task from 'true-myth/task';

import { listChecks, type ListChecksError, type ListChecksOptions } from './client';
import type { ConformanceCheckRecord } from './record';

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
		title: record.title,
		// Section is not on the v1 catalog row; category grouping uses the test id.
		section: '',
		sources: []
	};
}

export type ListFcafTestsOptions = Omit<ListChecksOptions, 'standard'>;

/**
 * List FCAF tests from the shared conformance catalog (not static codegen).
 */
export function listFcafTests(
	options: ListFcafTestsOptions = {}
): Task.Task<FCAFTestCatalogEntry[], ListChecksError> {
	return listChecks({ ...options, standard: FCAF_STANDARD }).map((records) =>
		records.map(toFcafCatalogEntry)
	);
}

/**
 * Promise helper for loaders / TanStack queryFn.
 */
export async function getFcafTests(
	options: ListFcafTestsOptions = {}
): Promise<FCAFTestCatalogEntry[] | Error> {
	const result = await listFcafTests(options);
	if (result.isErr) return result.error;
	return result.value;
}
