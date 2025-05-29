// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z, type ZodTypeAny, type ZodRawShape } from 'zod';
import { pipe, Record, Tuple } from 'effect';
import {
	fieldValueTypeSchema,
	stringifiedObjectSchema,
	type FieldConfig,
	type TestsConfigsFields
} from './tests-configs-form/types';

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

export function createTestListInputSchema(fields: TestsConfigsFields) {
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
