// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createCachedFetchLoad } from '$lib/utils/cached-fetch-load.js';

import { pb } from '@/pocketbase';

import type { HubItem } from './types.js';

/**
 * Id → hub_items record cache. Concurrent `get`s for the same id share one
 * lookup. Failures are not sticky so transient errors (e.g. 429) can be retried.
 */
const hubItemByIdCache = createCachedFetchLoad<string, HubItem>({
	lookup: async (id, fetchFn) => {
		try {
			return await pb.collection('hub_items').getOne<HubItem>(id, {
				fetch: fetchFn,
				requestKey: null
			});
		} catch (cause) {
			throw cause instanceof Error ? cause : new Error('Failed to get hub item by id');
		}
	}
});

export async function getHubItemById(id: string, options = { fetch }): Promise<HubItem> {
	return hubItemByIdCache.get(id, options);
}

/** Clears cached hub_items lookups. Intended for tests. */
export function invalidateHubItemByIdCache(): Promise<void> {
	return hubItemByIdCache.invalidateAll();
}
