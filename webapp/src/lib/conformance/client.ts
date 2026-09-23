// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { FCAFTestCatalogEntry } from '$lib/fcaf/catalog.js';

import { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { ZodError } from 'zod';

import { pb } from '@/pocketbase';

import { nestSuites } from './nest';
import {
	compileCheckListQuery,
	compileSuiteListQuery,
	type CatalogFacets,
	type CheckListIntent,
	type SuiteFacets,
	type SuiteListIntent
} from './query';
import {
	CONFORMANCE_CHECKS_COLLECTION,
	CONFORMANCE_SUITES_COLLECTION,
	conformanceCheckRecordSchema,
	conformanceSuiteRecordSchema,
	type ConformanceCheckRecord,
	type ConformanceSuiteRecord,
	type TemplateSurface
} from './record';
import { standardSchema, type Standard } from './types';

export type { CatalogFacets, SuiteFacets } from './query';

export type ListChecksError = ClientResponseError | ZodError;
export type ListSuitesError = ClientResponseError | ZodError;

export type ListChecksOptions = CheckListIntent & {
	fetch?: typeof fetch;
};

export type ListSuitesOptions = SuiteListIntent & {
	fetch?: typeof fetch;
};

/**
 * Shared conformance catalog client: flat checks, suite rows, nested browse
 * tree, and FCAF listing — PocketBase adapter over compiled domain intents.
 */
export function listChecks(
	options: ListChecksOptions = {}
): Task.Task<ConformanceCheckRecord[], ListChecksError> {
	const { fetch: fetchFn = fetch, ...intent } = options;
	const compiled = compileCheckListQuery(intent, (raw, params) => pb.filter(raw, params));

	const listOptions: {
		fetch: typeof fetch;
		sort: string;
		filter?: string;
	} = {
		fetch: fetchFn,
		sort: compiled.sort,
		...(compiled.filter ? { filter: compiled.filter } : {})
	};

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() => pb.collection(CONFORMANCE_CHECKS_COLLECTION).getFullList(listOptions)
	).andThen((rows) => {
		const parsed = conformanceCheckRecordSchema.array().safeParse(rows);
		if (parsed.success) return Task.resolve(parsed.data);
		return Task.reject(parsed.error);
	});
}

/**
 * Suite-grain catalog for the hub table.
 * Default sort intent: wallet→issuer→verifier, then standard, then suite uid.
 */
export function listSuites(
	options: ListSuitesOptions = {}
): Task.Task<ConformanceSuiteRecord[], ListSuitesError> {
	const { fetch: fetchFn = fetch, ...intent } = options;
	const compiled = compileSuiteListQuery(intent, (raw, params) => pb.filter(raw, params));

	const listOptions: {
		fetch: typeof fetch;
		sort: string;
		filter?: string;
	} = {
		fetch: fetchFn,
		sort: compiled.sort,
		...(compiled.filter ? { filter: compiled.filter } : {})
	};

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() => pb.collection(CONFORMANCE_SUITES_COLLECTION).getFullList(listOptions)
	).andThen((rows) => {
		const parsed = conformanceSuiteRecordSchema.array().safeParse(rows);
		if (parsed.success) return Task.resolve(parsed.data);
		return Task.reject(parsed.error);
	});
}

// --- Browse tree (nested standards → versions → suites) ---

export type ListAllResponse = Standard[];
export type ListAllError = ClientResponseError | ZodError;
export type StandardsWithTestSuites = ListAllResponse;

export type ListAllOptions = {
	fetch?: typeof fetch;
	surface?: TemplateSurface;
	facets?: SuiteFacets;
};

/**
 * Nested standards tree for hub, start-checks, and pipeline pickers.
 * Source: PocketBase `conformance_suites` (grouped on fs_* axes, ADR-0002).
 */
export function listAll(options: ListAllOptions = {}): Task.Task<ListAllResponse, ListAllError> {
	const { fetch: fetchFn = fetch, surface = 'manual', facets } = options;

	return listSuites({ fetch: fetchFn, surface, facets }).andThen((records) => {
		const nested = nestSuites(records);
		const res = standardSchema.array().safeParse(nested);
		if (res.success) return Task.resolve(res.data);
		return Task.reject(res.error);
	});
}

/** Promise-shaped helper for SvelteKit loaders (hub detail, start-checks). */
export async function getStandardsWithTestSuites(
	options: ListAllOptions = {}
): Promise<StandardsWithTestSuites | Error> {
	const result = await listAll(options);
	if (result.isErr) return result.error;
	return result.value;
}

// --- FCAF catalog listing ---

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

/** List FCAF tests from the shared conformance catalog (not static codegen). */
export function listFcafTests(
	options: ListFcafTestsOptions = {}
): Task.Task<FCAFTestCatalogEntry[], ListChecksError> {
	return listChecks({ ...options, standard: FCAF_STANDARD }).map((records) =>
		records.map(toFcafCatalogEntry)
	);
}

/** Promise helper for loaders / TanStack queryFn. */
export async function getFcafTests(
	options: ListFcafTestsOptions = {}
): Promise<FCAFTestCatalogEntry[] | Error> {
	const result = await listFcafTests(options);
	if (result.isErr) return result.error;
	return result.value;
}
