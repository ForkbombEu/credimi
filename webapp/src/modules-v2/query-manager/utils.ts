// import type { BaseTypes, CollectionName, MaybeArray, QueryOptions } from './types';

// export function applyRootOptions<T extends BaseTypes, C extends CollectionName<T>>(
// 	options: QueryOptions<T, C>,
// 	rootOptions: QueryOptions<T, C>
// ): QueryOptions<T, C> {
// 	return {
// 		sort: options.sort ?? rootOptions.sort,
// 		searchFields: options.searchFields ?? rootOptions.searchFields,
// 		search: options.search ?? rootOptions.search,
// 		perPage: options.perPage ?? rootOptions.perPage,
// 		filter: [...ensureArray(options.filter), ...ensureArray(rootOptions.filter)],
// 		excludeIDs: [...ensureArray(options.excludeIDs), ...ensureArray(rootOptions.excludeIDs)]
// 	};
// }

// export function ensureArray<T>(value: MaybeArray<T> | undefined): T[] {
// 	if (!value) return [];
// 	return Array.isArray(value) ? value : [value];
// }
