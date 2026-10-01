// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createCachedFetchLoad } from '$lib/utils/cached-fetch-load.js';

import { pb } from '@/pocketbase';

//

type GetRecordByCanonifiedPathResult<T = unknown> = { message: string; record?: T };

/**
 * Path → record cache. Concurrent `get`s for the same path share one lookup.
 * Failures are not sticky so transient errors (e.g. 429) can be retried.
 */
const pathRecordCache = createCachedFetchLoad<string, unknown>({
	lookup: async (path, fetchFn) => {
		let result: GetRecordByCanonifiedPathResult;
		try {
			result = await pb.send<GetRecordByCanonifiedPathResult>(
				'/api/canonify/identifier/validate',
				{
					method: 'POST',
					body: { canonified_name: path },
					fetch: fetchFn,
					requestKey: null
				}
			);
		} catch {
			throw new Error('Failed to get record by path');
		}
		if (result.record) {
			return result.record;
		}
		throw new Error(result.message);
	}
});

export async function getRecordByCanonifiedPath<T = unknown>(
	path: string,
	options = { fetch }
): Promise<T | Error> {
	const result = await pathRecordCache.getOrError(path, options);
	return result as T | Error;
}

/** Clears cached path lookups. Intended for tests. */
export function invalidateCanonifyPathCache(): Promise<void> {
	return pathRecordCache.invalidateAll();
}
