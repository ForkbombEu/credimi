// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export * from './collections-models.generated';

//

import { Array, Option, pipe } from 'effect';

import {
	CollectionsModels,
	type CollectionName,
	type AnyCollectionField,
	type FileCollectionField,
	type RelationCollectionField,
	type SelectCollectionField,
	type AnyCollectionModel
} from './collections-models.generated';

//

export function getCollectionModel(collection: CollectionName): AnyCollectionModel {
	return pipe(
		CollectionsModels as AnyCollectionModel[],
		Array.findFirst((model) => model.name == collection),
		Option.getOrThrowWith(() => new CollectionNotFoundError())
	);
}

export function getCollectionNameFromId(id: string): CollectionName {
	return pipe(
		CollectionsModels,
		Array.findFirst((model) => model.id == id),
		Option.getOrThrowWith(() => new CollectionNotFoundError()),
		(model) => model.name as CollectionName
	);
}

class CollectionNotFoundError extends Error {}

//

export function isArrayField(
	fieldConfig: AnyCollectionField
): fieldConfig is FileCollectionField | SelectCollectionField | RelationCollectionField {
	const type = fieldConfig.type;
	if (type !== 'select' && type !== 'relation' && type !== 'file') return false;
	if (fieldConfig.maxSelect === 1) return false;
	else return true;
}

export function getRelationFields<C extends CollectionName>(collection: C): string[] {
	return getCollectionModel(collection)
		.fields.filter((field) => field.type == 'relation')
		.map((field) => field.name);
}
