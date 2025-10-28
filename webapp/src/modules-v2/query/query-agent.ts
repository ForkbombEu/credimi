import type { AnyQuery, QueryResponseItem } from './types';

export interface QueryAgent<Q extends AnyQuery> {
	execute: () => Promise<QueryResponseItem<Q>>;
}
