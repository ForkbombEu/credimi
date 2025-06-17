// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';
import { jsonStringSchema } from '$lib/utils';

//

export const configFieldTypeSchema = z.literal('string').or(z.literal('object'));

export const baseConfigFieldSchema = z.object({
	CredimiID: z.string(),
	DescriptionKey: z.string(),
	LabelKey: z.string(),
	Type: configFieldTypeSchema,
	Example: z.string().optional()
});

export type BaseConfigField = z.infer<typeof baseConfigFieldSchema>;

export const namedConfigFieldSchema = baseConfigFieldSchema.extend({
	FieldName: z.string()
});

export type NamedConfigField = z.infer<typeof namedConfigFieldSchema>;

export type ConfigField = BaseConfigField | NamedConfigField;

//

export const checksConfigFieldsResponseSchema = z.object({
	normalized_fields: z.array(baseConfigFieldSchema),
	specific_fields: z.record(
		z.string(),
		z.object({
			content: jsonStringSchema,
			fields: z.array(namedConfigFieldSchema)
		})
	)
});

export type ChecksConfigFieldsResponse = z.infer<typeof checksConfigFieldsResponseSchema>;
