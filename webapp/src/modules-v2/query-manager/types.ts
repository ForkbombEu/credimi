import type { NonEmptyTuple, Simplify } from 'type-fest';

import type { CollectionName } from '@/pocketbase/collections-models';
import type { CollectionExpands, CollectionResponses } from '@/pocketbase/types';
import type { KeyOf } from '@/utils/types';

/* Main */

export type Query<C extends CollectionName, E extends ExpandOption<C> = never> = BaseQuery<C, E> &
	Partial<QueryOptions> &
	Partial<QueryParams<C>>;

export type QueryResult<C extends CollectionName, E extends ExpandOption<C> = never> = {
	query: Query<C, E>;
	records: QueryResponseItem<C, E>[];
	totalItems: number;
};

/*
 * Base query
 */

export type BaseQuery<C extends CollectionName, E extends ExpandOption<C> = never> = {
	collection: C;
	expand?: E;
};

// Expand

export type ExpandOption<C extends CollectionName> =
	Expand<C> extends Record<string, never> ? never : NonEmptyTuple<ExpandKey<C>>;

type Expand<C extends CollectionName> = CollectionExpands[C];
type ExpandKey<C extends CollectionName> = KeyOf<Expand<C>>;

type ResolvedExpand<C extends CollectionName, E extends ExpandOption<C> = never> = E extends never
	? never
	: Simplify<Pick<Expand<C>, E[number]>>;

// Response

export type QueryResponseItem<C extends CollectionName, E extends ExpandOption<C> = never> =
	ResolvedExpand<C, E> extends never
		? CollectionResponses[C]
		: Simplify<CollectionResponses[C] & { expand: ResolvedExpand<C, E> }>;

/*
 * Query parameters (variable)
 */

export type QueryParams<C extends CollectionName> = {
	page: number;
	perPage: number;
	filter: FilterParam;
	sort: SortParam<C>;
	search: SearchParam<C>;
	excludeIDs: ExcludeParam;
};

// Sort

export type SortParam<C extends CollectionName> = MaybeArray<[Field<C>, SortOrder]>;

type SortOrder = 'ASC' | 'DESC';

// Filter

export type FilterParam = MaybeArray<string | CompoundFilter>;

type FilterMode = 'OR' | 'AND';

type CompoundFilter = { id: string; expressions: string[]; mode: FilterMode };
// Sometimes we need to update the filter expression from the UI, so we need to keep the id

// Search

export type SearchParam<C extends CollectionName> = [text: string, fields: MaybeArray<Field<C>>];

// Exclude

export type ExcludeParam = MaybeArray<string>;

/*
 * Query options
 */

type QueryOptions = {
	fetch: typeof fetch;
	requestKey: string | null;
	url: URL;
};

/*
 * Utils
 */

type Field<C extends CollectionName> = KeyOf<CollectionResponses[C]>;
type MaybeArray<T> = T | T[];
