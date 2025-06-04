// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';
import { getExceptionMessage } from '@/utils/errors';
import { Record as R } from 'effect';
import {
	type ConfigField,
	namedConfigFieldSchema,
	type NamedConfigField
} from '$start-checks-form/types';

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

//

export interface BaseForm {
	getFormData(): Record<string, unknown>;
	isValid: boolean;
}

//

export const DEFAULT_INDENTATION = 2;

export function formatJson(json: string, indentation: number = DEFAULT_INDENTATION) {
	try {
		const parsed = JSON.parse(json);
		return JSON.stringify(parsed, null, indentation);
	} catch {
		return json;
	}
}

//

export function isNamedConfigField(field: ConfigField): field is NamedConfigField {
	return namedConfigFieldSchema.safeParse(field).success;
}

export function configFieldComparator(a: ConfigField, b: ConfigField) {
	// First compare by type - string comes before object
	if (a.Type !== b.Type) {
		return a.Type === 'string' ? -1 : 1;
	}
	// Then compare by name
	return a.CredimiID.localeCompare(b.CredimiID);
}
