// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ValueOf } from 'type-fest';

import type { CollectionName } from '@/pocketbase/collections-models';
import type { CollectionResponses } from '@/pocketbase/types';
import type { GenericRecord } from '@/utils/types';

import { getCollectionModel, getCollectionNameFromId } from '@/pocketbase/collections-models';

//

export function createRecordDisplayFromFields<R extends GenericRecord>(
	record: R,
	displayFields: (keyof R)[],
	separator = '–'
): string {
	return displayFields
		.map((f) => record[f])
		.filter((f) => Boolean(f))
		.join(` ${separator} `);
}

export type RecordPresenter<R> = (record: R) => string;

export function createDefaultRecordPresenter<C extends CollectionName, R = CollectionResponses[C]>(
	collection: C
): RecordPresenter<R> {
	const fields = getCollectionModel(collection)
		.fields.filter((field) => field.type == 'text')
		.map((field) => field.name) as (keyof R)[];

	return (record) => {
		return fields.map((f) => record[f]).join(' - ');
	};
}

export function createRecordDisplay<R extends ValueOf<CollectionResponses>>(
	record: R,
	displayFields: (keyof R)[] | undefined,
	displayFn: RecordPresenter<R> | undefined
) {
	if (displayFields) return createRecordDisplayFromFields(record, displayFields);
	else if (displayFn) return displayFn(record);
	else {
		const presenter = createDefaultRecordPresenter(
			getCollectionNameFromId(record.collectionId)
		);
		return presenter(record);
	}
}
