import type { Simplify } from 'type-fest';

import type {
	ConfigValuesExpand,
	ConfigValuesResponse,
	CredentialIssuersExpand,
	CredentialIssuersResponse
} from '@/pocketbase/types';

import type { AnyQuery, BaseQuery, ResolvedExpand } from './types';

import { createQueryManagerFor } from './query-manager.svelte';

type Types = {
	credentialIssuers: {
		type: CredentialIssuersResponse;
		expand: CredentialIssuersExpand;
	};
	configValues: {
		type: ConfigValuesResponse;
		expand: ConfigValuesExpand;
	};
	noExpand: {
		type: { value: string };
	};
};

export type QueryResponseItem<Q extends AnyQuery> =
	Q extends BaseQuery<infer T, infer C, infer E>
		? ResolvedExpand<T, C, E> extends never
			? T[C]['type']
			: Simplify<T[C]['type'] & { expand: ResolvedExpand<T, C, E> }>
		: never;

const MyTypesQueryManager = createQueryManagerFor<Types>();

const query = new MyTypesQueryManager({
	collection: 'credentialIssuers',
	expand: ['owner', 'credentials_via_credential_issuer'],
	sort: ['owner', 'ASC']
});

const records = query.records;
