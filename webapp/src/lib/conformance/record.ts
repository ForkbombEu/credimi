// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

import {
	conformanceCheckRecordSchema,
	conformanceSuiteRecordSchema,
	type ConformanceCheckRecord,
	type ConformanceSuiteRecord
} from './record.schemas';

/** Synthetic PocketBase collection name used in URLs only.
 * Data is served by Credimi from an ephemeral catalog (no data.db collection).
 * `generate.collections-models.ts` stubs CollectionName; `generate:catalog-pb-types`
 * injects CollectionRecords/Responses after pocketbase-typegen.
 */
export const CONFORMANCE_CHECKS_COLLECTION = 'conformance_checks' as const;

export const templateSurfaceSchema = z.enum(['manual', 'pipeline']);
export type TemplateSurface = z.infer<typeof templateSurfaceSchema>;

export {
	conformanceCheckRecordSchema,
	conformanceSuiteRecordSchema,
	type ConformanceCheckRecord,
	type ConformanceSuiteRecord
};

/** Suite-grain catalog collection name used in URLs only. */
export const CONFORMANCE_SUITES_COLLECTION = 'conformance_suites' as const;
