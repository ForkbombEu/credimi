// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { yamlStringSchema } from '$lib/utils';
import { z } from 'zod';

//

export const configFieldTypeSchema = z.literal('string').or(z.literal('object'));

export const baseConfigFieldSchema = z.object({
	credimi_id: z.string(),
	field_default_value: z.string().optional(),
	field_description: z.string(),
	field_label: z.string(),
	field_type: configFieldTypeSchema,
	field_options: z.array(z.string())
});

export type BaseConfigField = z.infer<typeof baseConfigFieldSchema>;

export const namedConfigFieldSchema = baseConfigFieldSchema.extend({
	field_id: z.string()
});

export type NamedConfigField = z.infer<typeof namedConfigFieldSchema>;

export type ConfigField = BaseConfigField | NamedConfigField;

//

export const checksConfigFieldsResponseSchema = z.object({
	normalized_fields: z.array(baseConfigFieldSchema),
	specific_fields: z.record(
		z.string(),
		z.object({
			content: yamlStringSchema,
			fields: z.array(namedConfigFieldSchema)
		})
	)
});

export type ChecksConfigFieldsResponse = z.infer<typeof checksConfigFieldsResponseSchema>;

//

export type { StartCheckResult, StartChecksResponse } from './response-types';
