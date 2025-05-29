// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z, type ZodTypeAny, type ZodRawShape } from 'zod';
import { getExceptionMessage } from '@/utils/errors';
import { pipe, Record as R, Record, Tuple } from 'effect';
import { pb } from '@/pocketbase';

/* Types & Schemas */

const fieldValueTypeSchema = z.literal('string').or(z.literal('object'));

const sharedFieldSchema = z.object({
	CredimiID: z.string(),
	DescriptionKey: z.string(),
	LabelKey: z.string(),
	Type: fieldValueTypeSchema,
	Example: z.string().optional()
});

export type FieldConfig = z.infer<typeof sharedFieldSchema>;

const specificFieldSchema = sharedFieldSchema.extend({
	FieldName: z.string()
});

export type SpecificFieldConfig = z.infer<typeof specificFieldSchema>;

//

export const stringifiedObjectSchema = z.string().superRefine((v, ctx) => {
	try {
		z.record(z.string(), z.unknown())
			.refine((value) => R.size(value) > 0)
			.parse(JSON.parse(v));
	} catch (e) {
		const message = getExceptionMessage(e);
		ctx.addIssue({
			code: z.ZodIssueCode.custom,
			message: `Invalid JSON object: ${message}`
		});
	}
});

export function createTestVariablesFormSchema(fields: FieldConfig[]) {
	const schemaRawShape: ZodRawShape = Object.fromEntries(
		fields.map((f) => {
			let schema: ZodTypeAny;
			if (f.Type == 'string') {
				schema = z.string().nonempty();
			} else if (f.Type == 'object') {
				schema = stringifiedObjectSchema;
			} else {
				throw new Error(`Invalid field type: ${f.Type}`);
			}
			return [f.CredimiID, schema];
		})
	);
	return z.object(schemaRawShape);
}

//

const fieldsResponseSchema = z.object({
	normalized_fields: z.array(sharedFieldSchema),
	specific_fields: z.record(
		z.string(),
		z.object({
			content: stringifiedObjectSchema,
			fields: z.array(specificFieldSchema)
		})
	)
});

export type FieldsResponse = z.infer<typeof fieldsResponseSchema>;

export async function getVariables(suiteAndVersionPath: string, filenames: string[]) {
	const data = await pb.send('/api/template/placeholders', {
		method: 'POST',
		body: {
			test_id: suiteAndVersionPath,
			filenames
		}
	});
	return fieldsResponseSchema.parse(data);
}

//

export const jsonTestInputSchema = z.object({
	format: z.literal('json'),
	data: stringifiedObjectSchema
});

export const variablesTestInputSchema = z.object({
	format: z.literal('variables'),
	data: z.record(
		z.string(),
		z.object({
			type: fieldValueTypeSchema,
			value: z.string().or(stringifiedObjectSchema),
			fieldName: z.string()
		})
	)
});

export const testInputSchema = jsonTestInputSchema.or(variablesTestInputSchema);

export type TestInput = z.infer<typeof testInputSchema>;

export function createTestListInputSchema(fields: FieldsResponse) {
	return z.object(Record.map(fields.specific_fields, () => testInputSchema));
}

//

export function createInitialDataFromFields(fields: FieldConfig[], excludeIds: string[] = []) {
	return pipe(
		fields
			.map((field) => {
				let example: string;
				if (field.Type == 'string') {
					example = field.Example ?? '';
				} else if (field.Type == 'object' && field.Example) {
					example = JSON.stringify(JSON.parse(field.Example), null, 4);
				} else {
					throw new Error(`Invalid field type: ${field.Type}`);
				}
				return Tuple.make(field.CredimiID, example);
			})
			.filter(([, value]) => value !== undefined && Boolean(value))
			.filter(([id]) => !excludeIds.includes(id)),
		(entries) => Object.fromEntries(entries)
	);
}

/*  */
