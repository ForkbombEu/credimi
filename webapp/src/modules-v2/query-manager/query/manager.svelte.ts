import type { CollectionName } from '@/pocketbase/collections-models';
import type * as query from './types';

//

type Init<C extends CollectionName, E extends query.ExpandOption<C> = never> =
	| query.Response<C, E>
	| query.Query<C, E>;

export class Manager<C extends CollectionName, E extends query.ExpandOption<C> = never> {
	constructor(private init: Init<C, E>) {}
}
