// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CollectionFormOptions } from '@/collections-components/form';
import type { IconComponent } from '@/components/types';
import type { CollectionName } from '@/pocketbase/collections-models';
import type { CollectionResponses } from '@/pocketbase/types';
import type { GenericRecord } from '@/utils/types';
import type { Snippet } from 'svelte';

//

export type TriggerProp = {
	button?: Snippet<
		[{ openForm: () => void; triggerAttributes: GenericRecord; icon: IconComponent }]
	>;
};

export type TitleProp = { formTitle?: string };

export type RecordProp<C extends CollectionName> = { record: CollectionResponses[C] };

export type RecordCreateEditProps<C extends CollectionName> = {
	collection?: C;
	buttonText?: Snippet;
} & CollectionFormOptions<C> &
	TriggerProp &
	TitleProp;

export type RecordAction = 'delete' | 'share' | 'edit' | 'select';
export type GlobalRecordAction = RecordAction | 'create';

export type HideOption = Array<RecordAction> | 'all';
