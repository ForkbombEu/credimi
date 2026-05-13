// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

const standardMetadataSchema = z.object({
	uid: z.string(),
	name: z.string(),
	description: z.string(),
	standard_url: z.string(),
	latest_update: z.string(),
	external_links: z.record(z.string(), z.array(z.string())).nullable(),
	disabled: z.boolean().optional()
});

const versionMetadataSchema = z.object({
	uid: z.string(),
	name: z.string(),
	latest_update: z.string(),
	specification_url: z.string().optional()
});

const suiteMetadataSchema = z.object({
	uid: z.string(),
	name: z.string(),
	homepage: z.string(),
	repository: z.string(),
	help: z.string(),
	description: z.string(),
	logo: z.string().optional()
});

export const suiteSchema = suiteMetadataSchema.extend({
	files: z.array(z.string()),
	paths: z.array(z.string())
});

export const versionSchema = versionMetadataSchema.extend({
	suites: z.array(suiteSchema)
});

export const standardSchema = standardMetadataSchema.extend({
	versions: z.array(versionSchema)
});

export type Suite = z.infer<typeof suiteSchema>;
export type Version = z.infer<typeof versionSchema>;
export type Standard = z.infer<typeof standardSchema>;
