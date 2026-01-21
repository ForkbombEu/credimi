// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { capitalize } from 'effect/String';
import _, { camelCase } from 'lodash';
import assert from 'node:assert';
import fs from 'node:fs/promises';
import path from 'node:path';

import {
	CollectionsModels,
	isArrayField,
	type AnyCollectionField,
	type AnyCollectionModel,
	type RelationCollectionField
} from '@/pocketbase/collections-models';
import { EXPORT_TYPE, formatCode, GENERATED, logCodegenResult, SEPARATOR } from '@/utils/codegen';

/* CONSTS */

const COLLECTION_RESPONSES = 'CollectionResponses';
const COLLECTION = 'Collection';

const FORM_DATA = 'FormData';
const EXPAND = 'Expand';
const ZOD_RAW_SHAPE = 'ZodRawShape';
const RELATED_COLLECTIONS = 'RelatedCollections';

const RECORD_NEVER = 'Record<string, never>';

const IMPORT_STATEMENTS = `
import type { ${COLLECTION_RESPONSES} } from '@/pocketbase/types/index.generated'
import type {z} from 'zod/v3'
`;

/* Functions */

class UnhandledFieldTypeError extends Error {}

main();

async function main() {
	const sortedCollections = _.sortBy(CollectionsModels, (d) => d.name);

	const formDataTypes = sortedCollections.map(createCollectionFormDataType);
	const formDataIndexType = createIndexType(formDataTypes, FORM_DATA);

	const expandTypes = sortedCollections.map(createCollectionExpand);
	const expandIndexType = createIndexType(expandTypes, EXPAND, true);

	const zodTypes = sortedCollections.map(createCollectionZodRawType);
	const zodIndexType = createIndexType(zodTypes, ZOD_RAW_SHAPE, true);

	const relatedCollectionTypes = sortedCollections.map(createCollectionRelatedCollections);
	const relatedCollectionsIndexType = createIndexType(
		relatedCollectionTypes,
		RELATED_COLLECTIONS
	);

	const code = [
		IMPORT_STATEMENTS,
		SEPARATOR,
		formDataIndexType,
		...formDataTypes.map((t) => t.code),
		SEPARATOR,
		zodIndexType,
		...zodTypes.map((t) => t.code),
		SEPARATOR,
		expandIndexType,
		...expandTypes.map((t) => t.code),
		SEPARATOR,
		relatedCollectionsIndexType,
		...relatedCollectionTypes.map((t) => t.code)
	].join('\n\n');

	const formattedCode = await formatCode(code);
	const filePath = path.join(import.meta.dirname, `extra.${GENERATED}.ts`);
	await fs.writeFile(filePath, formattedCode);
	logCodegenResult('extra types', filePath);
}

//

function createCollectionFormDataType(model: AnyCollectionModel): GeneratedCollectionTypeData {
	const collectionName = model.name;
	const typeName = capitalize(camelCase(model.name)) + FORM_DATA;

	const modelFields = (model.fields as AnyCollectionField[]).filter(
		(f) => f.name != 'id' && f.name != 'created' && f.name != 'updated'
	);

	const fields = modelFields.map((f) => {
		let type: string;
		if (f.type == 'number') type = 'number';
		else if (f.type == 'bool') type = 'boolean';
		else if (f.type == 'date') type = 'string';
		else if (f.type == 'editor') type = 'string';
		else if (f.type == 'email') type = 'string';
		else if (f.type == 'file') type = 'File';
		else if (f.type == 'json') type = 'unknown';
		else if (f.type == 'relation') type = 'string';
		else if (f.type == 'text') type = 'string';
		else if (f.type == 'url') type = 'string';
		else if (f.type == 'select') type = f.values.map((v) => `"${v}"`).join(' | ');
		else if (f.type == 'autodate') type = 'string';
		else if (f.type == 'password') type = 'string';
		else throw new UnhandledFieldTypeError();

		if (isArrayField(f)) type = `(${type})[]`;
		const optionalQuestionMark = f.required ? '' : '?';

		return `"${f.name}"${optionalQuestionMark} : ${type}`;
	});

	return {
		code: `${EXPORT_TYPE} ${typeName} = { ${fields.join('\n')} }`,
		typeName,
		collectionName
	};
}

// Needed for `@/pocketbase/zod-schema`

function createCollectionZodRawType(model: AnyCollectionModel): GeneratedCollectionTypeData {
	const collectionName = model.name;
	const typeName = capitalize(camelCase(model.name)) + ZOD_RAW_SHAPE;
	const modelFields = model.fields as AnyCollectionField[];

	const fields = modelFields
		.filter((f) => !['id', 'created', 'updated'].includes(f.name))
		.map((f) => {
			let type: string;
			if (f.type == 'number') type = 'z.ZodNumber';
			else if (f.type == 'bool') type = 'z.ZodBoolean';
			else if (f.type == 'date') type = 'z.ZodString';
			else if (f.type == 'editor') type = 'z.ZodString';
			else if (f.type == 'email') type = 'z.ZodString';
			else if (f.type == 'file') type = 'z.ZodType<File>';
			else if (f.type == 'json') type = 'z.ZodUnknown';
			else if (f.type == 'relation') type = 'z.ZodString';
			else if (f.type == 'select') type = `z.ZodEnum<${JSON.stringify(f.values)}>`;
			else if (f.type == 'text') type = 'z.ZodString';
			else if (f.type == 'url') type = 'z.ZodString';
			else if (f.type == 'autodate') type = 'z.ZodString';
			else if (f.type == 'password') type = 'z.ZodString';
			else throw new UnhandledFieldTypeError();

			if (isArrayField(f)) type = `z.ZodArray<${type}>`;
			if (!f.required) type = `z.ZodOptional<${type}>`;

			return `"${f.name}" : ${type}`;
		});

	return {
		code: `${EXPORT_TYPE} ${typeName} = { ${fields.join('\n')} }`,
		typeName,
		collectionName
	};
}

// Needed for `@/collections-components`

function createCollectionExpand(model: AnyCollectionModel): GeneratedCollectionTypeData {
	const collectionName = model.name;
	const typeName = capitalize(camelCase(model.name)) + EXPAND;

	const expands = [
		...createCollectionExpandItems(model),
		...createCollectionInverseExpandItems(model)
	];

	const expandCode = expands.length == 0 ? RECORD_NEVER : `{ ${expands.join('\n')} }`;

	return {
		code: `${EXPORT_TYPE} ${typeName} = ${expandCode}`,
		typeName,
		collectionName
	};
}

function createCollectionExpandItems(model: AnyCollectionModel): string[] {
	return model.fields
		.filter((field) => field.type == 'relation')
		.map((field) => {
			const model = CollectionsModels.find((m) => m.id == field.collectionId);
			assert(model, 'Missing model');
			const optionalQuestionMark = field.required ? '' : '?';
			//
			const optionalArray = field.maxSelect == 1 ? '' : '[]';
			// @ts-check
			return `${field.name}${optionalQuestionMark} : (${COLLECTION_RESPONSES}["${model.name}"])${optionalArray}`;
		});
}

function createCollectionInverseExpandItems(model: AnyCollectionModel): string[] {
	function isInverseRelationField(field: AnyCollectionField): field is RelationCollectionField {
		return field.type == 'relation' && field.collectionId == model.id;
	}

	const inverseRelatedCollections = CollectionsModels.filter((c) =>
		c.fields.some((f) => isInverseRelationField(f as AnyCollectionField))
	);

	return inverseRelatedCollections.flatMap((c) =>
		c.fields
			.filter((f) => isInverseRelationField(f as AnyCollectionField))
			.map((f) => `${c.name}_via_${f.name}?: ${COLLECTION_RESPONSES}["${c.name}"][]`)
	);
}

//

function createCollectionRelatedCollections(
	model: AnyCollectionModel
): GeneratedCollectionTypeData {
	const collectionName = model.name;
	const typeName = capitalize(camelCase(model.name)) + RELATED_COLLECTIONS;

	const relatedCollections = model.fields
		.filter((field) => field.type == 'relation')
		.map((field) => {
			const model = CollectionsModels.find((m) => m.id == field.collectionId);
			assert(model, 'missing model');
			return `${field.name} : "${model.name}"`;
		});

	const relatedType =
		relatedCollections.length == 0 ? RECORD_NEVER : `{ ${relatedCollections.join('\n')} }`;

	return {
		code: `${EXPORT_TYPE} ${typeName} = ${relatedType}`,
		typeName,
		collectionName
	};
}

//

function createIndexType(data: GeneratedCollectionTypeData[], category: string, addPlural = false) {
	const entries = data.map((d) => `${d.collectionName} : ${d.typeName}`);
	const s = addPlural ? 's' : '';
	return `${EXPORT_TYPE} ${COLLECTION}${category}${s} = { ${entries.join('\n')} }`;
}

//

type GeneratedCollectionTypeData = {
	code: string;
	collectionName: string;
	typeName: string;
};
