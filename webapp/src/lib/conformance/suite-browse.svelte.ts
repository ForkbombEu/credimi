// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { SortingState } from '@tanstack/table-core';

import { createQuery } from '@tanstack/svelte-query';

import type { CatalogSurface, ConformanceSuiteRecord } from './record';

import { listHubSuites } from './client';
import {
	isHubDefaultSuiteSort,
	SUITE_FACET_KEYS,
	suiteSortFromTableColumns,
	type SuiteFacets
} from './query';

//

export type SuiteFacetKey = (typeof SUITE_FACET_KEYS)[number];

export type SuiteBrowseProps = {
	/** Catalog surface — required (ADR-0008); hub table passes `pipeline`. */
	surface: CatalogSurface;
	/** SSR suite rows (used when sort/search/facets are default). */
	get initialSuites(): ConformanceSuiteRecord[];
	/** Debounced text search across suite display/identity fields. */
	get search(): string;
};

/** Empty facet filter state (all axes unset). */
export function emptySuiteFacetFilters(): Record<SuiteFacetKey, string> {
	return {
		standard: '',
		component: '',
		version: '',
		provider: ''
	};
}

/** Non-empty facet values from the hub filter UI. */
export function activeSuiteFacetsFromFilters(
	filters: Record<SuiteFacetKey, string>
): SuiteFacets {
	const facets: SuiteFacets = {};
	for (const key of SUITE_FACET_KEYS) {
		const value = filters[key];
		if (value) facets[key] = value;
	}
	return facets;
}

/** Distinct non-empty values for facet selects (stable from unfiltered set). */
export function distinctSuiteFacetValues(
	records: readonly ConformanceSuiteRecord[],
	key: SuiteFacetKey
): string[] {
	const values: string[] = [];
	for (const record of records) {
		const value = record[key];
		if (value.length > 0 && !values.includes(value)) values.push(value);
	}
	return values.sort((a, b) => a.localeCompare(b));
}

/**
 * Use SSR rows when hub-default sort, no search, no facets, and SSR payload present.
 * Catalog query is disabled in that case.
 */
export function shouldUseSSRSuiteBrowse(options: {
	sort: ReturnType<typeof suiteSortFromTableColumns>;
	searchQuery: string;
	hasActiveFilters: boolean;
	initialSuitesLength: number;
}): boolean {
	return (
		isHubDefaultSuiteSort(options.sort) &&
		options.searchQuery === '' &&
		!options.hasActiveFilters &&
		options.initialSuitesLength > 0
	);
}

/**
 * Product-axis suite browse for the hub table: filter/sort/search state, SSR gate,
 * facet-options fetch, and filtered catalog query. Not a Filesystem nest projection
 * (that remains {@link ./store.svelte.ts}).
 *
 * Public interface: props (incl. Catalog surface), sorting/filters state, and
 * derived browse outputs. TanStack queries and SSR policy stay private.
 *
 * Queries are created in the constructor after `props` is assigned — class field
 * initializers run before parameter properties, and `createQuery` eagerly reads
 * derived search/SSR state.
 */
export class SuiteBrowse {
	private readonly props: SuiteBrowseProps;

	/** Empty = no UI sort indicator; data still arrives in hub-default order. */
	sorting = $state<SortingState>([]);

	filters = $state<Record<SuiteFacetKey, string>>(emptySuiteFacetFilters());

	private readonly sortIntent = $derived(suiteSortFromTableColumns(this.sorting));
	private readonly searchQuery = $derived.by(() => this.props.search.trim());

	private readonly activeFacets = $derived.by((): SuiteFacets =>
		activeSuiteFacetsFromFilters(this.filters)
	);

	readonly hasActiveFilters = $derived(Object.keys(this.activeFacets).length > 0);

	private readonly useSSR = $derived.by(() =>
		shouldUseSSRSuiteBrowse({
			sort: this.sortIntent,
			searchQuery: this.searchQuery,
			hasActiveFilters: this.hasActiveFilters,
			initialSuitesLength: this.props.initialSuites.length
		})
	);

	private readonly facetOptionsQuery: ReturnType<
		typeof createQuery<ConformanceSuiteRecord[], Error>
	>;
	private readonly catalogQuery: ReturnType<typeof createQuery<ConformanceSuiteRecord[], Error>>;

	constructor(props: SuiteBrowseProps) {
		this.props = props;
		const surface = props.surface;

		this.facetOptionsQuery = createQuery(() => ({
			queryKey: ['conformance-suites', surface, 'hub', 'facet-options'] as const,
			queryFn: async () => {
				const result = await listHubSuites({ surface });
				if (result.isErr) throw result.error;
				return result.value;
			}
		}));

		this.catalogQuery = createQuery(() => {
			const sort = this.sortIntent;
			const q = this.searchQuery;
			const facets = this.activeFacets;

			return {
				queryKey: ['conformance-suites', surface, 'hub', sort, q, facets] as const,
				enabled: !this.useSSR,
				queryFn: async () => {
					const result = await listHubSuites({
						surface,
						sort,
						search: q || undefined,
						facets: Object.keys(facets).length > 0 ? facets : undefined
					});
					if (result.isErr) throw result.error;
					return result.value;
				}
			};
		});
	}

	private readonly facetSourceSuites = $derived.by((): ConformanceSuiteRecord[] => {
		if (this.facetOptionsQuery.data) return this.facetOptionsQuery.data;
		if (this.props.initialSuites.length > 0) return this.props.initialSuites;
		return [];
	});

	readonly facetOptions = $derived.by((): Record<SuiteFacetKey, string[]> => {
		const options = {} as Record<SuiteFacetKey, string[]>;
		for (const key of SUITE_FACET_KEYS) {
			options[key] = distinctSuiteFacetValues(this.facetSourceSuites, key);
		}
		return options;
	});

	readonly displayedSuites = $derived.by((): ConformanceSuiteRecord[] => {
		if (this.useSSR) return this.props.initialSuites;
		return this.catalogQuery.data ?? [];
	});

	readonly isLoading = $derived.by(() => this.catalogQuery.isFetching);

	clearFilters = () => {
		this.filters = emptySuiteFacetFilters();
	};
}
