// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { SuperForm } from 'sveltekit-superforms';

import { getContext, setContext } from 'svelte';

import type { CollectionName } from '@/pocketbase/collections-models';
import type { CollectionFormData, CollectionRecords } from '@/pocketbase/types';

//

const COLLECTION_FORM_CONTEXT = Symbol('collection-form-context');

export type CollectionFormContext<C extends CollectionName> = () => {
	recordId?: string;
	initialData?: Partial<CollectionRecords[C]>;
	collectionName: string;
	form: SuperForm<CollectionFormData[C]>;
};

export function getCollectionFormContext<C extends CollectionName>() {
	return getContext<CollectionFormContext<C>>(COLLECTION_FORM_CONTEXT);
}

export function setCollectionFormContext<C extends CollectionName>(
	context: CollectionFormContext<C>
) {
	setContext(COLLECTION_FORM_CONTEXT, context);
}
