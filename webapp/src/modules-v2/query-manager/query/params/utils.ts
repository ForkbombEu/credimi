// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

import type { CollectionResponses } from '@/pocketbase/types';
import type { KeyOf } from '@/utils/types';

import { getCollectionModel, type CollectionName } from '@/pocketbase/collections-models';

import type { SortParam, SortParamItem } from './types';

//

export const DEFAULT_PER_PAGE = 25;
export const DEFAULT_PAGE = 1;

//

export type Field<C extends CollectionName> = KeyOf<CollectionResponses[C]>;

//

export function ensureSortArray(sort: SortParam<never> | undefined): SortParamItem<never>[] {
	if (!sort) {
		return [];
	} else if (sort.length == 2 && typeof sort[1] === 'string') {
		return [sort as never];
	} else {
		return sort as SortParamItem<never>[];
	}
}

export type MaybeArray<T> = T | T[];

export function maybeArray<Z extends z.ZodTypeAny>(schema: Z) {
	return z.union([schema, z.array(schema)]);
}

export function getSearchableFields<C extends CollectionName>(collection: C): Field<C>[] {
	const model = getCollectionModel(collection);
	return model.fields.map((f) => f.name) as Field<C>[];
}
