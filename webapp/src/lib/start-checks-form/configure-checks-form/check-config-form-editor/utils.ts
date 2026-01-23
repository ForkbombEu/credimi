// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ConfigField } from '$start-checks-form/types';

import { jsonStringSchema } from '$lib/utils';
import { formatJson } from '$start-checks-form/_utils';
import { Tuple, pipe } from 'effect';
import { ZodString, z, type ZodEffects } from 'zod/v3';

//

type CheckConfigFormValueSchema = ZodString | ZodEffects<ZodString>;

export function createCheckConfigFormSchema(fields: ConfigField[]) {
	const schemaRawShape: Record<string, CheckConfigFormValueSchema> = Object.fromEntries(
		fields.map((f) => {
			let schema: ZodString | ZodEffects<ZodString>;
			if (f.field_type == 'string') {
				schema = z.string().nonempty();
			} else if (f.field_type == 'object') {
				schema = jsonStringSchema;
			} else {
				throw new Error(`Invalid field type: ${f.field_type}`);
			}
			return [f.credimi_id, schema];
		})
	);
	return z.object(schemaRawShape);
}

export function createCheckConfigFormInitialData(fields: ConfigField[], excludeIds: string[] = []) {
	return pipe(
		fields
			.map((field) => {
				let example: string;
				if (field.field_type == 'string') {
					example = field.field_default_value ?? '';
				} else if (field.field_type == 'object' && field.field_default_value) {
					example = formatJson(field.field_default_value);
				} else {
					throw new Error(`Invalid field type: ${field.field_type}`);
				}
				return Tuple.make(field.credimi_id, example);
			})
			.filter(([, value]) => value !== undefined && Boolean(value))
			.filter(([id]) => !excludeIds.includes(id)),
		(entries) => Object.fromEntries(entries)
	);
}
