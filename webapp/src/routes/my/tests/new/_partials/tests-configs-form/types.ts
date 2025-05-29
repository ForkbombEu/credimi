// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';
import { getExceptionMessage } from '@/utils/errors';
import { Record as R } from 'effect';

// Single field schema

export const fieldValueTypeSchema = z.literal('string').or(z.literal('object'));

const configFieldSchema = z.object({
	CredimiID: z.string(),
	DescriptionKey: z.string(),
	LabelKey: z.string(),
	Type: fieldValueTypeSchema,
	Example: z.string().optional()
});

export type ConfigField = z.infer<typeof configFieldSchema>;

const specificFieldSchema = configFieldSchema.extend({
	FieldName: z.string()
});

export type SpecificFieldConfig = z.infer<typeof specificFieldSchema>;

// Utility schemas

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

// API response schema

export const testsConfigsFieldsSchema = z.object({
	normalized_fields: z.array(configFieldSchema),
	specific_fields: z.record(
		z.string(),
		z.object({
			content: stringifiedObjectSchema,
			fields: z.array(specificFieldSchema)
		})
	)
});

export type TestsConfigsFields = z.infer<typeof testsConfigsFieldsSchema>;
