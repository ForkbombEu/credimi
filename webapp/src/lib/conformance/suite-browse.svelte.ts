// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { SortingState } from '@tanstack/table-core';

import { createQuery, keepPreviousData } from '@tanstack/svelte-query';

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

/** Distinct non-empty values for facet selects. */
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
 * Suites matching every active facet except `exceptKey` (for cascading option lists).
 * The excluded axis stays free so its select still lists all values available under
 * the other constraints.
 */
export function suitesMatchingOtherFacets(
	records: readonly ConformanceSuiteRecord[],
	filters: Record<SuiteFacetKey, string>,
	exceptKey: SuiteFacetKey
): ConformanceSuiteRecord[] {
	return records.filter((record) => {
		for (const key of SUITE_FACET_KEYS) {
			if (key === exceptKey) continue;
			const selected = filters[key];
			if (selected && record[key] !== selected) return false;
		}
		return true;
	});
}

/**
 * Per-axis facet options from the unfiltered suite set, narrowed by every *other*
 * active facet (classic faceted-search cascading).
 */
export function dynamicSuiteFacetOptions(
	records: readonly ConformanceSuiteRecord[],
	filters: Record<SuiteFacetKey, string>
): Record<SuiteFacetKey, string[]> {
	const options = {} as Record<SuiteFacetKey, string[]>;
	for (const key of SUITE_FACET_KEYS) {
		options[key] = distinctSuiteFacetValues(
			suitesMatchingOtherFacets(records, filters, key),
			key
		);
	}
	return options;
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
 * Keep the previous catalog page (or SSR seed) visible while a new search/filter
 * fetch is in flight — avoids flashing an empty table/empty state.
 */
export function suiteCatalogPlaceholderData(
	previousData: ConformanceSuiteRecord[] | undefined,
	initialSuites: readonly ConformanceSuiteRecord[]
): ConformanceSuiteRecord[] | undefined {
	const kept = keepPreviousData(previousData);
	if (kept !== undefined) return kept;
	return initialSuites.length > 0 ? [...initialSuites] : undefined;
}

/**
 * Product-axis suite browse for the hub table: filter/sort/search state, SSR gate,
 * cascading facet options (narrowed by other active facets), and filtered catalog
 * query. Not a Filesystem nest projection (that remains {@link ./store.svelte.ts}).
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
			const initialSuites = this.props.initialSuites;

			return {
				queryKey: ['conformance-suites', surface, 'hub', sort, q, facets] as const,
				enabled: !this.useSSR,
				placeholderData: (previousData: ConformanceSuiteRecord[] | undefined) =>
					suiteCatalogPlaceholderData(previousData, initialSuites),
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

		$effect(() => {
			this.pruneUnavailableFilters();
		});
	}

	private readonly facetSourceSuites = $derived.by((): ConformanceSuiteRecord[] => {
		if (this.facetOptionsQuery.data) return this.facetOptionsQuery.data;
		if (this.props.initialSuites.length > 0) return this.props.initialSuites;
		return [];
	});

	readonly facetOptions = $derived.by((): Record<SuiteFacetKey, string[]> =>
		dynamicSuiteFacetOptions(this.facetSourceSuites, this.filters)
	);

	/** provider slug → short label from suite rows (providers.yaml projected). */
	readonly providerLabels = $derived.by((): Record<string, string> => {
		const labels: Record<string, string> = {};
		for (const row of this.facetSourceSuites) {
			const slug = row.provider?.trim();
			const label = row.provider_label?.trim();
			if (slug && label && labels[slug] == null) {
				labels[slug] = label;
			}
		}
		return labels;
	});

	readonly displayedSuites = $derived.by((): ConformanceSuiteRecord[] => {
		if (this.useSSR) return this.props.initialSuites;
		return this.catalogQuery.data ?? [];
	});

	readonly isLoading = $derived.by(() => this.catalogQuery.isFetching);

	clearFilters = () => {
		this.filters = emptySuiteFacetFilters();
	};

	/**
	 * Drop facet selections that are no longer available under the other active
	 * filters (keeps selects coherent after cascading option changes).
	 */
	private pruneUnavailableFilters = () => {
		const options = this.facetOptions;
		let next: Record<SuiteFacetKey, string> | undefined;
		for (const key of SUITE_FACET_KEYS) {
			const selected = this.filters[key];
			if (selected && !options[key].includes(selected)) {
				if (!next) next = { ...this.filters };
				next[key] = '';
			}
		}
		if (next) this.filters = next;
	};
}
