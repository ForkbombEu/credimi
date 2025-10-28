import type { NonEmptyTuple, Simplify } from 'type-fest';

//

export type BaseTypes = Record<
	string,
	{ type: object; expand?: Record<string, object | object[]> }
>;

export type CollectionName<T extends BaseTypes> = KeyOf<T>;

type Expand<T extends BaseTypes, C extends CollectionName<T>> = T[C]['expand'];
type ExpandKey<T extends BaseTypes, C extends CollectionName<T>> = KeyOf<Expand<T, C>>;

export type ExpandOption<T extends BaseTypes, C extends CollectionName<T>> =
	Expand<T, C> extends never ? never : NonEmptyTuple<ExpandKey<T, C>>;

export type ResolvedExpand<
	T extends BaseTypes,
	C extends CollectionName<T>,
	E extends ExpandOption<T, C> = never
> = E extends never ? never : Simplify<Pick<Expand<T, C>, E[number]>>;

export type BaseQuery<
	T extends BaseTypes,
	C extends CollectionName<T>,
	E extends ExpandOption<T, C> = never
> = {
	collection: C;
	expand?: E;
};

export type QueryOptions<T extends BaseTypes, C extends CollectionName<T>> = Partial<{
	sort: MaybeArray<SortItem<T, C>>;
	searchFields: MaybeArray<Field<T, C>>;
	search: string;
	perPage: number;
	filter: MaybeArray<FilterItem>;
	excludeIDs: MaybeArray<string>;
}>;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type AnyQuery = BaseQuery<any, any, any>;

export type QueryResponseItem<Q extends AnyQuery> =
	Q extends BaseQuery<infer T, infer C, infer E>
		? ResolvedExpand<T, C, E> extends never
			? T[C]['type']
			: Simplify<T[C]['type'] & { expand?: ResolvedExpand<T, C, E> }>
		: never;

//

export type MaybeArray<T> = T | Array<T>;
type KeyOf<T> = Extract<keyof T, string>;

type Field<T extends BaseTypes, C extends CollectionName<T>> = KeyOf<T[C]['type']>;

//

type SortOrder = 'ASC' | 'DESC';
type SortItem<T extends BaseTypes, C extends CollectionName<T>> = [Field<T, C>, SortOrder];

//

export type FilterMode = 'OR' | 'AND';

export type CompoundFilter = { id: string; expressions: string[]; mode: FilterMode };
// Sometimes we need to update the filter expression from the UI, so we need to keep the id

type FilterItem = string | CompoundFilter;

//
