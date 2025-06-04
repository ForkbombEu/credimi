// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z, type ZodEffects, ZodString } from 'zod';
import type { ConfigField } from '$start-checks-form/types';
import { formatJson } from '$start-checks-form/_utils';
import { Tuple } from 'effect';
import { pipe } from 'effect';
import { jsonStringSchema } from '$lib/utils';

//

type CheckConfigFormValueSchema = ZodString | ZodEffects<ZodString>;

export function createCheckConfigFormSchema(fields: ConfigField[]) {
	const schemaRawShape: Record<string, CheckConfigFormValueSchema> = Object.fromEntries(
		fields.map((f) => {
			let schema: ZodString | ZodEffects<ZodString>;
			if (f.Type == 'string') {
				schema = z.string().nonempty();
			} else if (f.Type == 'object') {
				schema = jsonStringSchema;
			} else {
				throw new Error(`Invalid field type: ${f.Type}`);
			}
			return [f.CredimiID, schema];
		})
	);
	return z.object(schemaRawShape);
}

export function createCheckConfigFormInitialData(fields: ConfigField[], excludeIds: string[] = []) {
	return pipe(
		fields
			.map((field) => {
				let example: string;
				if (field.Type == 'string') {
					example = field.Example ?? '';
				} else if (field.Type == 'object' && field.Example) {
					example = formatJson(field.Example);
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
