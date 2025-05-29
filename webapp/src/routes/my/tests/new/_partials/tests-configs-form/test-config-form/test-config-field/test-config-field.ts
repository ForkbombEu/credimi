// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

//

export const testConfigFieldTypeSchema = z.literal('string').or(z.literal('object'));

export const testConfigFieldSchema = z.object({
	CredimiID: z.string(),
	DescriptionKey: z.string(),
	LabelKey: z.string(),
	Type: testConfigFieldTypeSchema,
	Example: z.string().optional()
});

export type TestConfigField = z.infer<typeof testConfigFieldSchema>;

export const testConfigFieldSpecificSchema = testConfigFieldSchema.extend({
	FieldName: z.string()
});

export type TestConfigFieldSpecific = z.infer<typeof testConfigFieldSpecificSchema>;
