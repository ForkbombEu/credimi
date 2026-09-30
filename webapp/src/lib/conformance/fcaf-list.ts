// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import * as Task from 'true-myth/task';

import {
	awaitTask,
	listChecks,
	type ListChecksError,
	type ListChecksOptions
} from './client.js';
import type { ConformanceCheckRecord } from './record.js';

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

/**
 * FCAF check-grain listing use case (ADR-0010). Compiles `fs_standard: 'fcaf'`
 * and maps path → test id/title. Catalog surface remains required (ADR-0008).
 */
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
