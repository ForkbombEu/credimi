// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { ZodError } from 'zod';

import type { TemplateSurface } from './record';

import { listChecks } from './client';
import { nestChecks } from './nest';
import { standardSchema, type Standard } from './types';

//

export type { TemplateSurface } from './record';

export type ListAllResponse = Standard[];
export type ListAllError = ClientResponseError | ZodError;
export type StandardsWithTestSuites = ListAllResponse;

/**
 * Nested standards tree for hub, start-checks, and pipeline pickers.
 * Source: PocketBase `conformance_checks` (grouped client-side).
 */
export function listAll(
	options: { fetch?: typeof fetch; surface?: TemplateSurface } = {}
): Task.Task<ListAllResponse, ListAllError> {
	const { fetch: fetchFn = fetch, surface = 'manual' } = options;

	return listChecks({ fetch: fetchFn, surface }).andThen((records) => {
		const nested = nestChecks(records);
		const res = standardSchema.array().safeParse(nested);
		if (res.success) return Task.resolve(res.data);
		return Task.reject(res.error);
	});
}

/**
 * Promise-shaped helper for SvelteKit loaders (hub layout, start-checks).
 */
export async function getStandardsWithTestSuites(
	options: { fetch?: typeof fetch; surface?: TemplateSurface } = {}
): Promise<StandardsWithTestSuites | Error> {
	const result = await listAll(options);
	if (result.isErr) return result.error;
	return result.value;
}
