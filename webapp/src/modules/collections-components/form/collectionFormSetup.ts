// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pipe, Record, String } from 'effect';
import { cloneDeep, merge } from 'lodash';
import { ClientResponseError, type CollectionModel } from 'pocketbase';
import { toast } from 'svelte-sonner';
import { setError, type FormPathLeaves, type SuperForm } from 'sveltekit-superforms';
import { zod } from 'sveltekit-superforms/adapters';
import z from 'zod/v3';

import type {
	CollectionName,
	FileCollectionField,
	SchemaFields
} from '@/pocketbase/collections-models';
import type {
	CollectionFormData,
	CollectionRecords,
	CollectionResponses
} from '@/pocketbase/types';
import type { GenericRecord } from '@/utils/types';

import { createForm, type FormOptions } from '@/forms';
import { m } from '@/i18n';
import { pb } from '@/pocketbase';
import { getCollectionModel } from '@/pocketbase/collections-models';
import { createCollectionZodSchema } from '@/pocketbase/zod-schema';
import { getExceptionMessage } from '@/utils/errors';
import { ensureArray } from '@/utils/other';

import type { CollectionFormProps } from './collectionFormTypes';

//

export function setupCollectionForm<C extends CollectionName>({
	collection,
	recordId,
	initialData = {},
	onSuccess = () => {},
	fieldsOptions = {},
	superformsOptions = {},
	uiOptions = {},
	beforeSubmit,
	refineSchema = (schema) => schema
}: CollectionFormProps<C>): SuperForm<CollectionFormData[C]> {
	const { exclude = [], include = [], defaults = {}, hide = {} } = fieldsOptions;
	const { toastText } = uiOptions;

	/* */

	const collectionModel = getCollectionModel(collection) as CollectionModel;
	const collectionFieldNames = collectionModel.fields.map((f) => f.name);

	/* Schema creation */
	// When "include" is specified it takes precedence: only those fields are in the schema.
	const includeSet = new Set(include as string[]);
	const effectiveExclude =
		include.length > 0
			? collectionFieldNames.filter((name) => !includeSet.has(name))
			: exclude;

	const baseSchema = refineSchema(createCollectionZodSchema(collection)) as z.AnyZodObject;
	const schema =
		include.length > 0
			? baseSchema.pick(Object.fromEntries(include.map((key) => [key, true])))
			: baseSchema.omit(Object.fromEntries(effectiveExclude.map((key) => [key, true])));

	/* Initial data processing */
	/* This must be done for two reasons
	 *
	 * 1. File fields
	 *
	 * Form expects a file,
	 * but file data coming from PocketBase is a string
	 *
	 * We solve it this way:
	 * -  Store the original initial data
	 * -  Convert the strings to "placeholder" files
	 * -  When submitting the form, match the new files with the original filenames
	 *
	 * 2. JSON fields
	 *
	 * JSON fields come from the server as objects
	 * but we edit them on the client as strings
	 *
	 * -
	 *
	 * (Also, useful for seeding and cleaning data)
	 */

	const processedInitialData: Partial<CollectionFormData[C]> = pipe(
		initialData,
		(data) => removeExcessProperties(data, collectionModel, effectiveExclude), // Removes also "collectionId", "created", ...
		(data) => mockInitialDataFiles(data, collectionModel),
		(data) => merge(cloneDeep(data), defaults, hide),
		(data) => stringifyJsonFields(data, collectionModel)
	);

	/* Form creation */

	const form = createForm<GenericRecord>({
		adapter: zod(schema),
		initialData: processedInitialData,
		options: {
			dataType: 'form',
			...(superformsOptions as FormOptions)
		},

		onSubmit: async ({ form }) => {
			try {
				const data = pipe(
					cleanFormDataFiles(form.data, initialData, collectionModel),
					Record.map((v) => (v === undefined ? null : v)) // IMPORTANT!
				);

				let processedData = data as CollectionFormData[C];
				if (beforeSubmit) {
					processedData = await beforeSubmit(data as CollectionFormData[C]);
				}

				let record: CollectionResponses[C];
				if (recordId) {
					record = await pb
						.collection(collection)
						.update<CollectionResponses[C]>(recordId, processedData);
				} else {
					record = await pb
						.collection(collection)
						.create<CollectionResponses[C]>(processedData);
				}

				const showToast = uiOptions?.showToastOnSuccess ?? true;
				try {
					if (showToast) {
						const text = toastText
							? toastText
							: recordId
								? m.Record_updated_successfully()
								: m.Record_created_successfully();

						toast.success(text);
					}
				} catch (e) {
					console.error(e);
				}

				await onSuccess(record, recordId ? 'edit' : 'create');
			} catch (e) {
				if (e instanceof ClientResponseError) {
					const details = e.data.data as Record<
						FormPathLeaves<CollectionRecords[C]>,
						{ message: string; code: string }
					>;

					Record.toEntries(details).forEach(([path, data]) => {
						if (path in form.data) setError(form, path, data.message);
						else setError(form, `${path} - ${data.message}`);
					});

					setError(form, e.message);
				} else {
					setError(form, getExceptionMessage(e));
				}
			}
		}
	});

	//

	return form as unknown as SuperForm<CollectionFormData[C]>;
}

//

function removeExcessProperties<T extends GenericRecord>(
	recordData: T,
	collectionModel: CollectionModel,
	exclude: string[] = []
): Partial<T> {
	const collectionFields = collectionModel.fields.map((f) => f.name);
	return Record.filter(recordData, (v, k) => {
		const isRecordField = collectionFields.includes(k);
		const isNotExcluded = !exclude.includes(k);
		const hasValue = Boolean(v); // Sometimes useful
		return isRecordField && isNotExcluded && hasValue;
	}) as Partial<T>;
}

//

function mockInitialDataFiles<C extends CollectionName>(
	recordData: Partial<CollectionRecords[C]>,
	collectionModel: CollectionModel
) {
	return mapRecordDataByFieldType(
		recordData,
		collectionModel,
		'file',
		(fieldValue, fieldConfig) => {
			if (Array.isArray(fieldValue) && fieldValue.every(String.isString)) {
				return fieldValue.map((filename) => mockFile(filename, fieldConfig));
			} else if (String.isString(fieldValue)) {
				return mockFile(fieldValue, fieldConfig);
			} else {
				return fieldValue;
			}
		}
	) as Partial<CollectionFormData[C]>;
}

export function mockFile(
	filename: string,
	fileFieldConfig: Pick<FileCollectionField, 'mimeTypes'>
) {
	let fileOptions: FilePropertyBag | undefined = undefined;
	const mimeTypes = fileFieldConfig.mimeTypes;
	if (Array.isArray(mimeTypes) && mimeTypes.length > 0) {
		fileOptions = { type: mimeTypes[0] };
	}
	return new MockFile(filename, fileOptions);
}

export class MockFile extends File {
	constructor(filename: string, options: FilePropertyBag = {}) {
		super([], filename, options);
	}
}

export function removeMockFiles<T extends GenericRecord>(data: T): T {
	const clone: GenericRecord = {};
	for (const [key, value] of Object.entries(data)) {
		if (value instanceof MockFile) {
			continue;
		} else if (Array.isArray(value)) {
			const data = [];
			for (const item of value) {
				if (item instanceof MockFile) {
					continue;
				}
				data.push(item);
			}
			clone[key] = data;
		} else {
			clone[key] = value;
		}
	}
	return clone as T;
}

//

function stringifyJsonFields<T extends GenericRecord>(
	recordData: GenericRecord,
	collectionModel: CollectionModel
): T {
	return mapRecordDataByFieldType(recordData, collectionModel, 'json', (fieldValue) => {
		if (!fieldValue) return fieldValue;
		return JSON.stringify(fieldValue);
	}) as T;
}

//

export function cleanFormDataFiles(
	recordData: GenericRecord,
	initialData: GenericRecord,
	model: CollectionModel
) {
	const data = cloneDeep(recordData);

	const initialDataFileFields = pipe(
		initialData,
		Record.filter((_, fieldName) => {
			return Boolean(
				model.fields.find(
					(fieldConfig) => fieldConfig.name == fieldName && fieldConfig.type == 'file'
				)
			);
		}),
		Record.filter((v) => Array.isArray(v) || String.isString(v)), // Ensure filenames
		Record.map((v) => ensureArray(v)) // Ensuring array for easier checking
	);

	for (const [field, initialFilenames] of Object.entries(initialDataFileFields)) {
		const newFieldValue = data[field];

		if (newFieldValue === undefined || newFieldValue === null) {
			continue;
		}
		//
		else if (newFieldValue instanceof File) {
			const isFileOld = initialFilenames.includes(newFieldValue.name);
			if (isFileOld) delete data[field];
		}
		//
		else if (Array.isArray(newFieldValue) && newFieldValue.every((v) => v instanceof File)) {
			const allFilenames = newFieldValue.map((file) => file.name);
			const newFiles = newFieldValue.filter((file) => !initialFilenames.includes(file.name));
			const filesToRemove = initialFilenames.filter(
				(filename) => !allFilenames.includes(filename)
			);

			if (newFiles.length === 0) delete data[field];
			else data[field] = newFiles;

			if (filesToRemove.length > 0) data[`${field}-`] = filesToRemove;
		}
	}

	return data;
}

/* Utils */

class FieldConfigNotFound extends Error {}

function mapRecordDataByFieldType<T extends keyof SchemaFields>(
	recordData: GenericRecord,
	model: CollectionModel,
	fieldType: T,
	handler: (value: unknown, fieldConfig: SchemaFields[T]) => unknown
) {
	return pipe(
		recordData,
		Record.map((fieldValue, fieldName) => {
			const fieldConfig = model.fields.find((field) => field.name == fieldName);
			if (!fieldConfig) throw new FieldConfigNotFound();
			if (fieldConfig.type != fieldType) return fieldValue;
			return handler(fieldValue, fieldConfig as SchemaFields[T]);
		})
	);
}

//

export function removeEmptyValues(data: GenericRecord) {
	return Record.filter(data, (v) => {
		if (v === undefined || v === null) return false;
		if (typeof v == 'string') return String.isNonEmpty(v);
		return true;
	});
}
