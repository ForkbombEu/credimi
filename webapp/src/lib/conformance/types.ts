// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

/** Filesystem path-axis node: uid + display name only (not authored standard.yaml). */
const nestStandardSchema = z.object({
	uid: z.string(),
	name: z.string()
});

/** Filesystem path-axis node: uid + display name only (not authored version.yaml). */
const nestVersionSchema = z.object({
	uid: z.string(),
	name: z.string()
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

export const suiteMemberSchema = z.object({
	path: z.string(),
	title: z.string(),
	file: z.string()
});

export const suiteSchema = suiteMetadataSchema.extend({
	members: z.array(suiteMemberSchema)
});

export const versionSchema = nestVersionSchema.extend({
	suites: z.array(suiteSchema)
});

export const standardSchema = nestStandardSchema.extend({
	versions: z.array(versionSchema)
});

export type SuiteMember = z.infer<typeof suiteMemberSchema>;
export type Suite = z.infer<typeof suiteSchema>;
export type Version = z.infer<typeof versionSchema>;
export type Standard = z.infer<typeof standardSchema>;
