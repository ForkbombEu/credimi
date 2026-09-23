// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

/** Synthetic PocketBase collection name used in URLs only.
 * Data is served by Credimi from an ephemeral catalog (no data.db collection).
 * `generate.collections-models.ts` stubs CollectionName; `generate:catalog-pb-types`
 * injects CollectionRecords/Responses after pocketbase-typegen.
 */
export const CONFORMANCE_CHECKS_COLLECTION = 'conformance_checks' as const;

export const templateSurfaceSchema = z.enum(['manual', 'pipeline']);
export type TemplateSurface = z.infer<typeof templateSurfaceSchema>;

/** Flat catalog row projected by pkg/conformancecatalog (lean check; suite meta on suites). */
export const conformanceCheckRecordSchema = z.object({
	id: z.string(),
	path: z.string(),
	title: z.string(),
	standard: z.string(),
	version: z.string(),
	suite: z.string(),
	file: z.string(),
	visible_in: z.array(z.string()).optional().default([]),
	protocol: z.string().optional().default(''),
	sut: z.string().optional().default(''),
	role: z.string().optional().default(''),
	provider: z.string().optional().default(''),
	norm_standard: z.string().optional().default(''),
	component: z.string().optional().default(''),
	norm_version: z.string().optional().default('')
});

export type ConformanceCheckRecord = z.infer<typeof conformanceCheckRecordSchema>;

/** Suite-grain catalog row from `/api/collections/conformance_suites/records`. */
export const CONFORMANCE_SUITES_COLLECTION = 'conformance_suites' as const;

export const conformanceSuiteRecordSchema = z.object({
	id: z.string(),
	standard: z.string(),
	component: z.string().optional().default(''),
	component_rank: z.number().int().nonnegative().optional().default(9),
	version: z.string().optional().default(''),
	suite: z.string(),
	provider: z.string().optional().default(''),
	suite_name: z.string().optional().default(''),
	suite_homepage: z.string().optional().default(''),
	suite_repository: z.string().optional().default(''),
	suite_help: z.string().optional().default(''),
	suite_description: z.string().optional().default(''),
	suite_logo: z.string().optional().default(''),
	check_count: z.number().int().nonnegative(),
	check_paths: z.array(z.string()).optional().default([]),
	check_titles: z.array(z.string()).optional().default([]),
	check_files: z.array(z.string()).optional().default([]),
	visible_in: z.array(z.string()).optional().default([]),
	fs_standard: z.string(),
	fs_version: z.string(),
	path_prefix: z.string()
});

export type ConformanceSuiteRecord = z.infer<typeof conformanceSuiteRecordSchema>;
