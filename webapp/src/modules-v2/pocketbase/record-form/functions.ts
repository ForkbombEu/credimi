// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { db, pocketbase as pb } from '#';
import type { CollectionModel } from 'pocketbase';

import { Record, String } from 'effect';

import { getCollectionModel, type AnyCollectionField } from '@/pocketbase/collections-models';

//

export function recordToFormData<C extends db.CollectionName>(
	collection: C,
	data: Partial<pb.BaseRecord<C>>
): Partial<db.CollectionFormData[C]> {
	const model = getCollectionModel(collection) as CollectionModel;
	const fields = model.fields as AnyCollectionField[];
	const formData: Record<string, unknown> = {};

	for (const [fieldName, fieldValue] of Record.toEntries(data)) {
		const fieldConfig = fields.find((f) => f.name == fieldName);
		if (!fieldConfig) continue;

		// [file]
		if (fieldConfig.type == 'file') {
			const mimeType = fieldConfig.mimeTypes.at(0);
			if (Array.isArray(fieldValue) && fieldValue.every(String.isString)) {
				formData[fieldName] = fieldValue.map(
					(filename) => new MockFile(filename, { mimeType })
				);
			} else if (String.isString(fieldValue)) {
				formData[fieldName] = new MockFile(fieldValue, { mimeType });
			}
		}
		// [json]
		else if (fieldConfig.type == 'json') {
			formData[fieldName] = JSON.stringify(fieldValue);
		}
		// [rest]
		else {
			formData[fieldName] = fieldValue;
		}
	}

	return formData as Partial<db.CollectionFormData[C]>;
}

export class MockFile extends File {
	constructor(filename: string, options: { mimeType?: string }) {
		super([], filename, { type: options.mimeType });
	}
}

// //

// function removeExcessProperties<T extends GenericRecord>(
// 	recordData: T,
// 	collectionModel: CollectionModel,
// 	exclude: string[] = []
// ): Partial<T> {
// 	const collectionFields = collectionModel.fields.map((f) => f.name);
// 	return Record.filter(recordData, (v, k) => {
// 		const isRecordField = collectionFields.includes(k);
// 		const isNotExcluded = !exclude.includes(k);
// 		const hasValue = Boolean(v); // Sometimes useful
// 		return isRecordField && isNotExcluded && hasValue;
// 	}) as Partial<T>;
// }

// //

// //

// export function cleanFormDataFiles(
// 	recordData: GenericRecord,
// 	initialData: GenericRecord,
// 	model: CollectionModel
// ) {
// 	const data = cloneDeep(recordData);

// 	const initialDataFileFields = pipe(
// 		initialData,
// 		Record.filter((_, fieldName) => {
// 			return Boolean(
// 				model.fields.find(
// 					(fieldConfig) => fieldConfig.name == fieldName && fieldConfig.type == 'file'
// 				)
// 			);
// 		}),
// 		Record.filter((v) => Array.isArray(v) || String.isString(v)), // Ensure filenames
// 		Record.map((v) => ensureArray(v)) // Ensuring array for easier checking
// 	);

// 	for (const [field, initialFilenames] of Object.entries(initialDataFileFields)) {
// 		const newFieldValue = data[field];

// 		if (newFieldValue === undefined || newFieldValue === null) {
// 			continue;
// 		}
// 		//
// 		else if (newFieldValue instanceof File) {
// 			const isFileOld = initialFilenames.includes(newFieldValue.name);
// 			if (isFileOld) delete data[field];
// 		}
// 		//
// 		else if (Array.isArray(newFieldValue) && newFieldValue.every((v) => v instanceof File)) {
// 			const allFilenames = newFieldValue.map((file) => file.name);
// 			const newFiles = newFieldValue.filter((file) => !initialFilenames.includes(file.name));
// 			const filesToRemove = initialFilenames.filter(
// 				(filename) => !allFilenames.includes(filename)
// 			);

// 			if (newFiles.length === 0) delete data[field];
// 			else data[field] = newFiles;

// 			if (filesToRemove.length > 0) data[`${field}-`] = filesToRemove;
// 		}
// 	}

// 	return data;
// }

// /* Utils */

// class FieldConfigNotFound extends Error {}

// function mapRecordDataByFieldType<T extends keyof SchemaFields>(
// 	recordData: GenericRecord,
// 	model: CollectionModel,
// 	fieldType: T,
// 	handler: (value: unknown, fieldConfig: SchemaFields[T]) => unknown
// ) {
// 	return pipe(
// 		recordData,
// 		Record.map((fieldValue, fieldName) => {
// 			const fieldConfig = model.fields.find((field) => field.name == fieldName);
// 			if (!fieldConfig) throw new FieldConfigNotFound();
// 			if (fieldConfig.type != fieldType) return fieldValue;
// 			return handler(fieldValue, fieldConfig as SchemaFields[T]);
// 		})
// 	);
// }

// //

// export function removeEmptyValues(data: GenericRecord) {
// 	return Record.filter(data, (v) => {
// 		if (v === undefined || v === null) return false;
// 		if (typeof v == 'string') return String.isNonEmpty(v);
// 		return true;
// 	});
// }
