// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

import type { CollectionName } from '@/pocketbase/collections-models';
import type { CollectionResponses } from '@/pocketbase/types';
import type { KeyOf } from '@/utils/types';

import { SortOrderSchema, type SortParam, type SortParamItem } from './types';

//

export const DEFAULT_PER_PAGE = 25;
export const DEFAULT_PAGE = 1;

//

export type Field<C extends CollectionName> = KeyOf<CollectionResponses[C]>;

//

export function ensureSortArray(sort: SortParam<never> | undefined): SortParamItem<never>[] {
	if (!sort) {
		return [];
	} else if (sort.length == 2 && SortOrderSchema.safeParse(sort[1]).success) {
		return [sort as never];
	} else {
		return sort as SortParamItem<never>[];
	}
}

export type MaybeArray<T> = T | T[];

export function maybeArray<Z extends z.ZodTypeAny>(schema: Z) {
	return z.union([schema, z.array(schema)]);
}
