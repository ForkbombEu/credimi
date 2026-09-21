// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import PocketBase, { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { ZodError } from 'zod';

import { pb } from '@/pocketbase';

import {
	CONFORMANCE_CHECKS_COLLECTION,
	conformanceCheckRecordSchema,
	type ConformanceCheckRecord,
	type TemplateSurface
} from './record';

export type ListChecksError = ClientResponseError | ZodError;

export type ListChecksOptions = {
	fetch?: typeof fetch;
	/** When set, keep only rows whose `visible_in` includes this surface. */
	surface?: TemplateSurface;
};

/**
 * Shared conformance catalog client: lists flat checks from
 * `pb.collection('conformance_checks')`. Nested pickers group client-side.
 */
export function listChecks(
	options: ListChecksOptions = {}
): Task.Task<ConformanceCheckRecord[], ListChecksError> {
	const { fetch: fetchFn = fetch, surface } = options;

	const listOptions: {
		fetch: typeof fetch;
		sort: string;
		filter?: string;
	} = {
		fetch: fetchFn,
		sort: 'standard,version,suite,path'
	};

	if (surface) {
		// Multi-select "contains" — matches blueprints surface filtering.
		listOptions.filter = pb.filter('visible_in ~ {:surface}', { surface });
	}

	// Collection may be missing from generated TypedPocketBase until typegen runs.
	const catalog = pb as PocketBase;

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() => catalog.collection(CONFORMANCE_CHECKS_COLLECTION).getFullList(listOptions)
	).andThen((rows) => {
		const parsed = conformanceCheckRecordSchema.array().safeParse(rows);
		if (parsed.success) return Task.resolve(parsed.data);
		return Task.reject(parsed.error);
	});
}
