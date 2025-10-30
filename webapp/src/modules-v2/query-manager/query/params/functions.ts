// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Array, Record } from 'effect';
import { build, parse } from 'search-params';

import type { CollectionName } from '@/pocketbase/collections-models';

import { ensureArray } from '@/utils/other';

import { QueryParamsSchema, type QueryParams } from './types';
import { ensureSortArray } from './utils';

//

export function serialize<C extends CollectionName>(params: QueryParams<C>): string {
	return build(QueryParamsSchema.encode(params as never));
}

export function deserialize<C extends CollectionName>(queryString: string): QueryParams<C> {
	return QueryParamsSchema.decode(parse(queryString)) as QueryParams<C>;
}

/**
 * Merges two sets of query params according to specific rules:
 * - filter fields from both params1 and params2 are combined with 'AND' logic.
 * - excludeIDs fields from both params1 and params2 are combined with 'AND' logic.
 * - search field is replaced: if params2 provides a search, it takes precedence; otherwise, params1's search is used.
 * - sort fields from both params1 and params2 are concatenated, the ones from 1 take precedence.
 *
 * @param params1 - The first set
 * @param params2 - The second set; fields in this take precedence in cases not merged as above
 * @returns A new set of query params that is the result of merging the two input sets by the above logic
 */

export function merge<C extends CollectionName>(
	params1: QueryParams<C>,
	params2: QueryParams<C>
): QueryParams<C> {
	const result: QueryParams<C> = {
		perPage: params2.perPage ?? params1.perPage,
		page: params2.page ?? params1.page,
		search: params2.search ?? params1.search,
		searchFields: params2.searchFields ?? params1.searchFields,
		sort: params2.sort ?? params1.sort
	};

	const exclude = [params1.excludeIDs, params2.excludeIDs].flatMap(ensureArray);
	if (Array.isNonEmptyArray(exclude)) result.excludeIDs = exclude;

	const filter = [params1.filter, params2.filter].flatMap(ensureArray);
	if (Array.isNonEmptyArray(filter)) result.filter = filter;

	const sort = [params1.sort, params2.sort].flatMap(ensureSortArray as never);
	if (Array.isNonEmptyArray(sort)) result.sort = sort as never;

	return Record.filter(result as never, Boolean);
}
