// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

//

export const testConfigFieldTypeSchema = z.literal('string').or(z.literal('object'));

export const baseTestConfigFieldSchema = z.object({
	CredimiID: z.string(),
	DescriptionKey: z.string(),
	LabelKey: z.string(),
	Type: testConfigFieldTypeSchema,
	Example: z.string().optional()
});

type BaseTestConfigField = z.infer<typeof baseTestConfigFieldSchema>;

export const namedTestConfigFieldSchema = baseTestConfigFieldSchema.extend({
	FieldName: z.string()
});

type NamedTestConfigField = z.infer<typeof namedTestConfigFieldSchema>;

export type TestConfigField = BaseTestConfigField | NamedTestConfigField;

//

export function isNamedTestConfigField(field: TestConfigField): field is NamedTestConfigField {
	return namedTestConfigFieldSchema.safeParse(field).success;
}
