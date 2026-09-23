// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pipe, Array as A, Record } from 'effect';
import fs from 'fs';
import 'dotenv/config';
import JsonToTS from 'json-to-ts';
import { capitalize, merge } from 'lodash';
import path from 'node:path';
import { type CollectionModel } from 'pocketbase';

import {
	EXPORT_TYPE,
	formatCode,
	GENERATED,
	logCodegenResult,
	SEPARATOR,
	openDb
} from '@/utils/codegen';

/* Constants */
const COLLECTION_FIELD = `CollectionField`;
const COLLECTION_MODEL = `CollectionModel`;
const ANY_COLLECTION_FIELD = `Any${COLLECTION_FIELD}`;
const ANY_COLLECTION_MODEL = `AnyCollectionModel`;

/* Setup */

// Define a type for collection fields
type CollectionField = {
	id: string;
	name: string;
	type: string;
	system: boolean;
	required?: boolean;
	options?: {
		values?: string[];
		[key: string]: unknown;
	};
	values?: string[];
	[key: string]: unknown;
};

async function getCollectionsFromDb(): Promise<CollectionModel[]> {
	const db = await openDb();
	const collections = await db.all(`SELECT * FROM _collections`);

	// Process each collection's fields which are stored as a JSON string
	for (const collection of collections) {
		try {
			// Fields are already stored as JSON in the collections table
			if (typeof collection.fields === 'string') {
				collection.fields = JSON.parse(collection.fields);
			}

			// Process each field to ensure proper structure
			collection.fields = collection.fields.map((field: CollectionField) => {
				// Make sure options is an object
				if (typeof field.options === 'string' && field.options) {
					try {
						field.options = JSON.parse(field.options);
					} catch (e) {
						console.warn(`Failed to parse options for field ${field.name}:`, e);
						field.options = {};
					}
				}

				// Ensure values are available for select fields
				if (
					field.type === 'select' &&
					field.options?.values &&
					Array.isArray(field.options.values)
				) {
					field.values = field.options.values;
				}

				return field;
			});
		} catch (e) {
			console.error(`Failed to process fields for collection ${collection.name}:`, e);
			collection.fields = [];
		}
	}

	await db.close();
	return collections as CollectionModel[];
}

/* Main */

async function main() {
	const models = await getCollectionsFromDb();
	// Synthetic: Credimi owns /api/collections/conformance_checks/* and
	// /api/collections/conformance_suites/* from an ephemeral :memory: cache;
	// there is no durable data.db collection.
	injectSyntheticConformanceChecks(models);
	injectSyntheticConformanceSuites(models);

	/* Codegen */

	const IMPORT_STATEMENTS = `
import type { ${COLLECTION_MODEL} } from 'pocketbase'
import type { SetFieldType, Simplify } from 'type-fest';
`;

	const schemaFieldTypes = getFieldTypeNames(models);
	const schemaFieldType = `${EXPORT_TYPE} SchemaFieldType = ${schemaFieldTypes.map((t) => JSON.stringify(t)).join(' | ')}`;
	const schemaFieldOptionsTypesData = schemaFieldTypes.map((f) =>
		createFieldOptionsTypeData(f, models)
	);
	const schemaFields = `export type SchemaFields = {
		${schemaFieldOptionsTypesData.map(({ name, key }) => `${key}: ${name}`).join('\n')}
	}`;

	const anySchemaField = `export type ${ANY_COLLECTION_FIELD} = ${schemaFieldOptionsTypesData.map(({ name }) => name).join(' | ')}`;

	const anyCollectionModel = `export type ${ANY_COLLECTION_MODEL} = Simplify<SetFieldType<${COLLECTION_MODEL}, 'schema', ${ANY_COLLECTION_FIELD}[]>>;`;

	const code = [
		IMPORT_STATEMENTS,
		SEPARATOR,
		schemaFieldType,
		anySchemaField,
		schemaFields,
		...schemaFieldOptionsTypesData.map((data) => data.code),
		SEPARATOR,
		anyCollectionModel,
		collectionName(models),
		sanitizeCollectionsModels(models)
	].join('\n\n');

	/* Export */

	const formattedCode = await formatCode(code);
	const filePath = path.resolve(import.meta.dirname, `collections-models.${GENERATED}.ts`);
	fs.writeFileSync(filePath, formattedCode);
	logCodegenResult('collections models and helper types', filePath);
}

// Execute the main function
main().catch(console.error);

/* Helper functions */

/** Stub so CollectionName includes the fake catalog URL collection after typegen. */
function injectSyntheticConformanceChecks(models: CollectionModel[]): void {
	if (models.some((m) => m.name === 'conformance_checks')) return;

	const text = (name: string, required = true): CollectionField => ({
		id: `conformance_checks_${name}`,
		name,
		type: 'text',
		system: false,
		required
	});

	models.push({
		id: 'pbc_conformance_checks_catalog',
		name: 'conformance_checks',
		type: 'base',
		system: false,
		fields: [
			{ id: 'conformance_checks_id', name: 'id', type: 'text', system: true, required: true },
			text('path'),
			text('title'),
			text('standard'),
			text('version'),
			text('suite'),
			text('file'),
			{
				id: 'conformance_checks_visible_in',
				name: 'visible_in',
				type: 'json',
				system: false,
				required: false
			},
			text('protocol', false),
			text('sut', false),
			text('role', false),
			text('provider', false),
			text('norm_standard', false),
			text('component', false),
			text('norm_version', false),
			text('suite_name', false),
			text('suite_homepage', false),
			text('suite_repository', false),
			text('suite_help', false),
			text('suite_description', false),
			text('suite_logo', false),
			{
				id: 'conformance_checks_created',
				name: 'created',
				type: 'autodate',
				system: false,
				required: false
			},
			{
				id: 'conformance_checks_updated',
				name: 'updated',
				type: 'autodate',
				system: false,
				required: false
			}
		]
	} as CollectionModel);
}

/** Stub so CollectionName includes the fake suites catalog URL collection. */
function injectSyntheticConformanceSuites(models: CollectionModel[]): void {
	if (models.some((m) => m.name === 'conformance_suites')) return;

	const text = (name: string, required = true): CollectionField => ({
		id: `conformance_suites_${name}`,
		name,
		type: 'text',
		system: false,
		required
	});

	models.push({
		id: 'pbc_conformance_suites_catalog',
		name: 'conformance_suites',
		type: 'base',
		system: false,
		fields: [
			{ id: 'conformance_suites_id', name: 'id', type: 'text', system: true, required: true },
			text('standard'),
			text('component', false),
			{
				id: 'conformance_suites_component_rank',
				name: 'component_rank',
				type: 'number',
				system: false,
				required: true
			},
			text('version', false),
			text('suite'),
			text('provider', false),
			text('suite_name', false),
			text('suite_homepage', false),
			text('suite_repository', false),
			text('suite_help', false),
			text('suite_description', false),
			text('suite_logo', false),
			{
				id: 'conformance_suites_check_count',
				name: 'check_count',
				type: 'number',
				system: false,
				required: true
			},
			{
				id: 'conformance_suites_check_paths',
				name: 'check_paths',
				type: 'json',
				system: false,
				required: false
			},
			{
				id: 'conformance_suites_check_titles',
				name: 'check_titles',
				type: 'json',
				system: false,
				required: false
			},
			{
				id: 'conformance_suites_check_files',
				name: 'check_files',
				type: 'json',
				system: false,
				required: false
			},
			{
				id: 'conformance_suites_visible_in',
				name: 'visible_in',
				type: 'json',
				system: false,
				required: false
			},
			text('fs_standard'),
			text('fs_version'),
			text('path_prefix'),
			{
				id: 'conformance_suites_created',
				name: 'created',
				type: 'autodate',
				system: false,
				required: false
			},
			{
				id: 'conformance_suites_updated',
				name: 'updated',
				type: 'autodate',
				system: false,
				required: false
			}
		]
	} as CollectionModel);
}

function sanitizeCollectionsModels(models: CollectionModel[]) {
	// Hiding API rules to reduce leaked information
	const sanitizedModels = models.map(
		Record.map((v, k) => {
			if (k.includes('Rule')) return '';
			else return v;
		})
	);

	return `export const CollectionsModels = ${JSON.stringify(sanitizedModels, null, 2)} as unknown as ${ANY_COLLECTION_MODEL}[]`;
}

function collectionName(models: CollectionModel[]): string {
	const names = models.map((m) => m.name);
	return `export type CollectionName = ${names.map((n) => JSON.stringify(n)).join(' | ')}`;
}

//

function getFieldTypeNames(models: CollectionModel[]) {
	return pipe(
		models.flatMap((model) => model.fields),
		A.map((field) => field.type),
		A.dedupe
	);
}

function createFieldOptionsTypeData(
	fieldType: string,
	models: CollectionModel[]
): GeneratedTypeData {
	const typeName = capitalize(fieldType) + COLLECTION_FIELD;
	return pipe(
		models.flatMap((m) => m.fields).filter((f) => f.type == fieldType),
		// merging data in a single object
		// somehow `required` is not present on some sytem fields, we add it here
		(fieldsSchemas) => merge({ required: false }, ...fieldsSchemas),
		// converting to ts
		(data) => JsonToTS(data, { useTypeAlias: true, rootName: typeName })[0],
		//
		(code) => {
			const newCode = code
				.replace('type: string;', `type: "${fieldType}";`)
				.replace('any[]', 'string[]');
			return `export ${newCode}`;
		},
		//
		(code) => ({
			code,
			name: typeName,
			key: fieldType
		})
	);
}

type GeneratedTypeData = {
	code: string;
	name: string;
	key: string;
};

//

export function pipeLog<T>(data: T): T {
	console.log(data);
	return data;
}
