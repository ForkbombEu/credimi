// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z, type ZodEffects, ZodString } from 'zod';
import type { TestConfigField } from '$lib/start-checks-form/test-config-field/test-config-field';
import { stringifiedObjectSchema } from '$lib/start-checks-form/_utils';
import { Tuple } from 'effect';
import { pipe } from 'effect';

//

type TestConfigFormValueSchema = ZodString | ZodEffects<ZodString>;

export function createTestConfigFormSchema(fields: TestConfigField[]) {
	const schemaRawShape: Record<string, TestConfigFormValueSchema> = Object.fromEntries(
		fields.map((f) => {
			let schema: ZodString | ZodEffects<ZodString>;
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

export function createTestConfigFormInitialData(
	fields: TestConfigField[],
	excludeIds: string[] = []
) {
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
