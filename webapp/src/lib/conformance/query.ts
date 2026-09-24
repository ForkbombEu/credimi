// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CatalogSurface } from './record';

/** Suite-grain hub facets (normalized product axes). */
export type SuiteFacets = {
	standard?: string;
	component?: string;
	version?: string;
	provider?: string;
};

/** Domain columns the hub suite table can sort by. */
export type SuiteSortColumn = 'component' | 'standard' | 'suite';

/**
 * Sort intent for suite listing. Callers never pass PocketBase field names
 * (`component_rank`, leading `-`); the compile step maps domain columns.
 */
export type SuiteSortIntent =
	| { kind: 'default' }
	| { kind: 'columns'; columns: ReadonlyArray<{ column: SuiteSortColumn; desc?: boolean }> };

/** Domain intent for listing suite-grain catalog rows. */
export type SuiteListIntent = {
	/** Catalog surface — required; no silent omit (ADR-0008 / ADR-0010). */
	surface: CatalogSurface;
	facets?: SuiteFacets;
	search?: string;
	sort?: SuiteSortIntent;
};

/**
 * Domain intent for listing check-grain catalog rows.
 * Production check use is FCAF listing (`fs_standard: 'fcaf'`); product facets
 * live on suite grain ({@link SuiteFacets}).
 */
export type CheckListIntent = {
	/** Catalog surface — required; no silent omit (ADR-0008 / ADR-0010). */
	surface: CatalogSurface;
	/** Filesystem standard uid (e.g. `fcaf`). */
	fs_standard?: string;
};

/** Adapter-ready list options after compiling a domain intent. */
export type CompiledListQuery = {
	filter?: string;
	sort: string;
};

/** Injected filter compiler (PocketBase `pb.filter` or a test stub). */
export type FilterCompiler = (raw: string, params?: Record<string, unknown>) => string;

/** Facet field order for suite-grain hub filters. */
export const SUITE_FACET_KEYS = ['standard', 'component', 'version', 'provider'] as const;

/** Hub table default: wallet→issuer→verifier, then standard, then suite uid. */
export const HUB_SUITE_SORT_DEFAULT: SuiteSortIntent = { kind: 'default' };

const HUB_SUITE_DEFAULT_SORT_STRING = 'component_rank,standard,suite';
const CHECK_DEFAULT_SORT_STRING = 'fs_standard,fs_version,suite,path';

/** PocketBase sort fields behind domain {@link SuiteSortColumn} values. */
const SUITE_SORT_PB_FIELD: Record<SuiteSortColumn, string> = {
	component: 'component_rank',
	standard: 'standard',
	suite: 'suite_name'
};

const SUITE_SORT_COLUMNS = new Set<string>(['component', 'standard', 'suite']);

/** True when sort is unset or the hub default preset. */
export function isHubDefaultSuiteSort(
	sort: SuiteSortIntent | undefined
): sort is undefined | { kind: 'default' } {
	return sort == null || sort.kind === 'default';
}

/**
 * Map TanStack / table column sort state into a suite sort intent.
 */
export function suiteSortFromTableColumns(
	columns: ReadonlyArray<{ id: string; desc: boolean }>
): SuiteSortIntent {
	const known = columns
		.filter((c) => SUITE_SORT_COLUMNS.has(c.id))
		.map((c) => ({
			column: c.id as SuiteSortColumn,
			...(c.desc ? { desc: true as const } : {})
		}));
	if (known.length === 0) return HUB_SUITE_SORT_DEFAULT;
	return { kind: 'columns', columns: known };
}

/**
 * Compile a suite list intent into filter/sort strings for a list adapter.
 */
export function compileSuiteListQuery(
	intent: SuiteListIntent,
	filterFn: FilterCompiler
): CompiledListQuery {
	const filters: string[] = [
		filterFn('visible_in ~ {:surface}', { surface: intent.surface })
	];
	appendSuiteFacetFilters(filters, intent.facets, filterFn);
	appendSuiteSearchFilter(filters, intent.search, filterFn);

	return {
		filter: filters.join(' && '),
		sort: compileSuiteSort(intent.sort)
	};
}

/**
 * Compile a check list intent into filter/sort strings for a list adapter.
 */
export function compileCheckListQuery(
	intent: CheckListIntent,
	filterFn: FilterCompiler
): CompiledListQuery {
	const filters: string[] = [
		filterFn('visible_in ~ {:surface}', { surface: intent.surface })
	];
	if (intent.fs_standard) {
		filters.push(filterFn('fs_standard = {:fs_standard}', { fs_standard: intent.fs_standard }));
	}

	return {
		filter: filters.join(' && '),
		sort: CHECK_DEFAULT_SORT_STRING
	};
}

function compileSuiteSort(sort: SuiteSortIntent | undefined): string {
	if (isHubDefaultSuiteSort(sort)) return HUB_SUITE_DEFAULT_SORT_STRING;

	return sort.columns
		.map(({ column, desc }) => {
			const field = SUITE_SORT_PB_FIELD[column];
			return desc ? `-${field}` : field;
		})
		.join(',');
}

function appendSuiteFacetFilters(
	filters: string[],
	facets: SuiteFacets | undefined,
	filterFn: FilterCompiler
): void {
	if (!facets) return;
	for (const key of SUITE_FACET_KEYS) {
		const value = facets[key];
		if (value == null || value === '') continue;
		filters.push(filterFn(`${key} = {:${key}}`, { [key]: value }));
	}
}

function appendSuiteSearchFilter(
	filters: string[],
	search: string | undefined,
	filterFn: FilterCompiler
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
