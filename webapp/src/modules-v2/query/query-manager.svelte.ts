import type {
	BaseQuery,
	BaseTypes,
	CollectionName,
	ExpandOption,
	QueryOptions,
	QueryResponseItem
} from './types';
import { applyRootOptions } from './utils';

//

class QueryManager<
	T extends BaseTypes,
	C extends CollectionName<T>,
	E extends ExpandOption<T, C> = never
> {
	public readonly records: QueryResponseItem<BaseQuery<T, C, E>>[] = $state([]);

	public readonly collection: C;
	private readonly expand: E | undefined;

	private readonly rootOptions: QueryOptions<T, C>;
	public readonly options: QueryOptions<T, C> = $state({});
	private mergedOptions: QueryOptions<T, C> = $derived.by(() =>
		applyRootOptions(this.options, this.rootOptions)
	);

	constructor(init: BaseQuery<T, C, E> & QueryOptions<T, C>) {
		const { collection, expand, ...rest } = init;
		this.collection = collection;
		this.expand = expand;
		this.rootOptions = rest;
	}
}

export type TypedQueryManager<T extends BaseTypes> = {
	new <C extends CollectionName<T>, E extends ExpandOption<T, C> = never>(
		init: BaseQuery<T, C, E> & QueryOptions<T, C>
	): QueryManager<T, C, E>;
};

export function createQueryManagerFor<T extends BaseTypes>(): TypedQueryManager<T> {
	return QueryManager as TypedQueryManager<T>;
}
