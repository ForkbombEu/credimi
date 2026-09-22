// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import PocketBase, { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { ZodError } from 'zod';

import { pb } from '@/pocketbase';

import {
	CONFORMANCE_CHECKS_COLLECTION,
	conformanceCheckRecordSchema,
	type ConformanceCheckRecord,
	type TemplateSurface
} from './record';

export type ListChecksError = ClientResponseError | ZodError;

export type CatalogFacets = {
	protocol?: string;
	sut?: string;
	role?: string;
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

/** Facet field order for PB equality filters (stable join order). */
export const CATALOG_FACET_KEYS = ['protocol', 'sut', 'role', 'provider'] as const;

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
 * Shared conformance catalog client: lists flat checks from the fake
 * PocketBase collection URL (`conformance_checks`). Nested pickers group client-side.
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
