// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

/** PocketBase `conformance_checks` collection name (catalog facade). */
export const CONFORMANCE_CHECKS_COLLECTION = 'conformance_checks' as const;

export const templateSurfaceSchema = z.enum(['manual', 'pipeline']);
export type TemplateSurface = z.infer<typeof templateSurfaceSchema>;

/** Flat catalog row projected by pkg/conformancecatalog. */
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
	provider: z.string().optional().default('')
});

export type ConformanceCheckRecord = z.infer<typeof conformanceCheckRecordSchema>;
