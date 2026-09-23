// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { FCAFTestCatalogEntry } from '$lib/fcaf/catalog.js';

import PocketBase, { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { ZodError } from 'zod';

import { pb } from '@/pocketbase';

import { nestSuites } from './nest';
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

export type ListChecksError = ClientResponseError | ZodError;
export type ListSuitesError = ClientResponseError | ZodError;

/** Check-grain facets (start-checks / legacy filters). */
export type CatalogFacets = {
	protocol?: string;
	sut?: string;
	role?: string;
	provider?: string;
};

/** Suite-grain hub facets (normalized product axes). */
export type SuiteFacets = {
	standard?: string;
	component?: string;
	version?: string;
	provider?: string;
};

export type ListChecksOptions = {
	fetch?: typeof fetch;
	/** When set, keep only rows whose `visible_in` includes this surface. */
	surface?: TemplateSurface;
	/** When set, keep only rows for this standard uid (e.g. `fcaf`). */
	standard?: string;
	/** Facet equality filters (PocketBase `field = value`). */
	facets?: CatalogFacets;
};

export type ListSuitesOptions = {
	fetch?: typeof fetch;
	surface?: TemplateSurface;
	facets?: SuiteFacets;
	sort?: string;
	/** PocketBase `~` match across suite display fields. */
	search?: string;
};

/** Facet field order for check-grain PB equality filters. */
export const CATALOG_FACET_KEYS = ['protocol', 'sut', 'role', 'provider'] as const;

/** Facet field order for suite-grain hub filters. */
export const SUITE_FACET_KEYS = ['standard', 'component', 'version', 'provider'] as const;

export type PbFilterFn = (raw: string, params?: Record<string, unknown>) => string;

/**
 * Append PocketBase equality filters for each set catalog facet.
 * Skips missing/empty values; preserves {@link CATALOG_FACET_KEYS} order.
 */
export function appendFacetFilters(
	filters: string[],
	facets: CatalogFacets | undefined,
	filterFn: PbFilterFn
): void {
	if (!facets) return;
	for (const key of CATALOG_FACET_KEYS) {
		const value = facets[key];
		if (value) {
			filters.push(filterFn(`${key} = {:${key}}`, { [key]: value }));
		}
	}
}

/**
 * Append a suite text-search filter (`~`) across display and identity fields.
 */
export function appendSuiteSearchFilter(
	filters: string[],
	search: string | undefined,
	filterFn: PbFilterFn
): void {
	const q = search?.trim();
	if (!q) return;
	filters.push(
		filterFn(
			'(suite_name ~ {:q} || suite ~ {:q} || standard ~ {:q} || component ~ {:q} || version ~ {:q} || provider ~ {:q})',
			{ q }
		)
	);
}

/**
 * Append suite-projection facet filters in {@link SUITE_FACET_KEYS} order.
 */
export function appendSuiteFacetFilters(
	filters: string[],
	facets: SuiteFacets | undefined,
	filterFn: PbFilterFn
): void {
	if (!facets) return;
	for (const key of SUITE_FACET_KEYS) {
		const value = facets[key];
		if (value) {
			filters.push(filterFn(`${key} = {:${key}}`, { [key]: value }));
		}
	}
}

/**
 * Shared conformance catalog client: flat checks, suite rows, nested browse
 * tree, and FCAF listing — all over the fake PocketBase collection URLs.
 */
export function listChecks(
	options: ListChecksOptions = {}
): Task.Task<ConformanceCheckRecord[], ListChecksError> {
	const { fetch: fetchFn = fetch, surface, standard, facets } = options;

	const listOptions: {
		fetch: typeof fetch;
		sort: string;
		filter?: string;
	} = {
		fetch: fetchFn,
		sort: 'standard,version,suite,path'
	};

	const filters: string[] = [];
	if (surface) {
		filters.push(pb.filter('visible_in ~ {:surface}', { surface }));
	}
	if (standard) {
		filters.push(pb.filter('standard = {:standard}', { standard }));
	}
	appendFacetFilters(filters, facets, (raw, params) => pb.filter(raw, params));
	if (filters.length > 0) {
		listOptions.filter = filters.join(' && ');
	}

	// Collection may be missing from generated TypedPocketBase until typegen runs.
	const catalog = pb as PocketBase;

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() => catalog.collection(CONFORMANCE_CHECKS_COLLECTION).getFullList(listOptions)
	).andThen((rows) => {
		const parsed = conformanceCheckRecordSchema.array().safeParse(rows);
		if (parsed.success) return Task.resolve(parsed.data);
		return Task.reject(parsed.error);
	});
}

/**
 * Suite-grain catalog for the hub table.
 * Default sort: wallet→issuer→verifier, then standard, then suite uid.
 */
export function listSuites(
	options: ListSuitesOptions = {}
): Task.Task<ConformanceSuiteRecord[], ListSuitesError> {
	const {
		fetch: fetchFn = fetch,
		surface,
		facets,
		sort = 'component_rank,standard,suite',
		search
	} = options;

	const listOptions: {
		fetch: typeof fetch;
		sort: string;
		filter?: string;
	} = {
		fetch: fetchFn,
		sort
	};

	const filters: string[] = [];
	if (surface) {
		filters.push(pb.filter('visible_in ~ {:surface}', { surface }));
	}
	appendSuiteFacetFilters(filters, facets, (raw, params) => pb.filter(raw, params));
	appendSuiteSearchFilter(filters, search, (raw, params) => pb.filter(raw, params));
	if (filters.length > 0) {
		listOptions.filter = filters.join(' && ');
	}

	const catalog = pb as PocketBase;

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() => catalog.collection(CONFORMANCE_SUITES_COLLECTION).getFullList(listOptions)
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

/** Promise-shaped helper for SvelteKit loaders (hub layout, start-checks). */
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
