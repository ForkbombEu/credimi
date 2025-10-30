// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Snippet } from 'svelte';

import type { CollectionFormOptions } from '@/collections-components/form';
import type { CollectionName } from '@/pocketbase/collections-models';
import type { PocketbaseQueryExpandOption } from '@/pocketbase/query';
import type { FilterMode } from '@/pocketbase/query/query.js';
import type { CollectionResponses } from '@/pocketbase/types/index.js';
import type { CollectionZodSchema } from '@/pocketbase/zod-schema/index.js';

import { setupDerivedContext } from '@/utils/svelte-context';

import { CollectionManager } from './collectionManager.svelte.js';

//

export type Filter = {
	name: string;
	expression: string;
};

export type FilterGroup = {
	name?: string;
	id: string;
	mode: FilterMode;
	filters: Filter[];
};

export type FiltersOption = FilterGroup | FilterGroup[];

//

export type CollectionManagerContext<
	C extends CollectionName = never,
	Expand extends PocketbaseQueryExpandOption<C> = never
> = {
	manager: CollectionManager<C, Expand>;
	filters: FiltersOption;
	formsOptions: Record<FormPropType, CollectionFormOptions<C>>;
	formRefineSchema: (schema: CollectionZodSchema<C>) => CollectionZodSchema<C>;

	createForm?: Snippet<[{ closeSheet: () => void }]>;
	editForm?: Snippet<[{ record: CollectionResponses[C]; closeSheet: () => void }]>;
};

type FormPropType = 'base' | 'create' | 'edit';

export const {
	getDerivedContext: getCollectionManagerContext,
	setDerivedContext: setCollectionManagerContext
} = setupDerivedContext<CollectionManagerContext>('cmc');
