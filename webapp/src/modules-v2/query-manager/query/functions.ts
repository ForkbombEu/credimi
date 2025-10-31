// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { RecordListOptions } from 'pocketbase';

import type { CollectionName } from '@/pocketbase/collections-models';

import { pb } from '@/pocketbase';

import type * as query from './types';

import * as params from './params';

//

export async function getOne<C extends CollectionName, E extends query.ExpandOption<C> = never>(
	query: query.BaseQuery<C, E> & query.Options & { id: string }
) {
	const record: query.ResponseRecord<C, E> = await pb
		.collection(query.collection)
		.getOne(query.id);
	return record;
}

export async function getList<C extends CollectionName, E extends query.ExpandOption<C> = never>(
	query: query.Query<C, E>
): Promise<query.Response<C, E>> {
	const { listOptions, rootParams, urlParams } = build(query);
	const list = await pb
		.collection(query.collection)
		.getList(listOptions.page, listOptions.perPage, listOptions);
	return {
		records: list.items as query.ResponseRecord<C, E>[],
		totalItems: list.totalItems,
		rootParams,
		urlParams,
		requestKey: listOptions.requestKey
	};
}

export function build<C extends CollectionName, E extends query.ExpandOption<C> = never>(
	query: query.Query<C, E>
) {
	const { collection, url, requestKey, fetch, expand, ...rest } = query;

	const rootParams: params.QueryParams<C> = rest;
	let urlParams: params.QueryParams<C> = {};
	if (url) urlParams = params.deserialize(url.searchParams.toString());

	const mergedParams = params.merge(rootParams, urlParams);
	if (!mergedParams.searchFields) {
		mergedParams.searchFields = params.getSearchableFields(collection);
	}

	const listOptions: RecordListOptions = {
		fetch,
		requestKey,
		...params.build(mergedParams)
	};

	if (expand) listOptions.expand = expand.join(',');

	return { listOptions, rootParams, urlParams, mergedParams };
}
