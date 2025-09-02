// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ControlAttrs } from 'formsnap';

import type { CollectionName } from '@/pocketbase/collections-models';
import type {
	PocketbaseQueryOptions,
	PocketbaseQueryResponse,
	PocketbaseQueryExpandOption
} from '@/pocketbase/query';
import type { CollectionRecords } from '@/pocketbase/types';

import type { RecordPresenter } from './utils';

//

export type CollectionInputRecordProps<
	C extends CollectionName,
	E extends PocketbaseQueryExpandOption<C> = never
> = {
	queryOptions?: Partial<PocketbaseQueryOptions<C, E>>;
	displayFields?: (keyof CollectionRecords[C])[] | undefined;
	displayFn?: RecordPresenter<PocketbaseQueryResponse<C, E>> | undefined;
};

export type CollectionInputProps<
	C extends CollectionName,
	E extends PocketbaseQueryExpandOption<C> = never
> = {
	collection: C;
	disabled?: boolean;
	label?: string | undefined;
	placeholder?: string | undefined;
	onSelect?: (record: PocketbaseQueryResponse<C, E>) => void;
	clearValueOnSelect?: boolean;
	controlAttrs?: ControlAttrs;
} & CollectionInputRecordProps<C, E>;
