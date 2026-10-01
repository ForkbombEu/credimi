// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Cache, Duration, Effect, Exit, FiberRef, GlobalValue } from 'effect';

import { pb } from '@/pocketbase';

import type { HubItem } from './types.js';

/**
 * Per-fiber fetch override so SSR can pass SvelteKit's `fetch` while the
 * shared id cache still dedupes concurrent lookups for the same hub item.
 *
 * @see https://effect.website/docs/caching/cache/
 */
const hubItemFetchRef = GlobalValue.globalValue(Symbol.for('@credimi/hub-item/fetch'), () =>
	FiberRef.unsafeMake<typeof fetch>(fetch)
);

/**
 * Id → hub_items record cache. Concurrent `get`s for the same id share one
 * lookup. Failures use TTL zero so transient errors (e.g. 429) are not sticky.
 */
const hubItemByIdCache = Effect.runSync(
	Cache.makeWith<string, HubItem, Error>({
		capacity: 2048,
		lookup: (id) =>
			Effect.gen(function* () {
				const fetchFn = yield* FiberRef.get(hubItemFetchRef);
				return yield* Effect.tryPromise({
					try: () =>
						pb.collection('hub_items').getOne<HubItem>(id, {
							fetch: fetchFn,
							requestKey: null
						}),
					catch: (cause) =>
						cause instanceof Error ? cause : new Error('Failed to get hub item by id')
				});
			}),
		timeToLive: (exit) => (Exit.isSuccess(exit) ? Duration.infinity : Duration.zero)
	})
);

export async function getHubItemById(id: string, options = { fetch }): Promise<HubItem> {
	return Effect.runPromise(
		hubItemByIdCache.get(id).pipe(
			Effect.catchAll((error) =>
				// Drop failed entries so transient errors (e.g. 429) can be retried.
				hubItemByIdCache.invalidate(id).pipe(Effect.andThen(Effect.fail(error)))
			),
			Effect.locally(hubItemFetchRef, options.fetch)
		)
	);
}

/** Clears cached hub_items lookups. Intended for tests. */
export function invalidateHubItemByIdCache(): Promise<void> {
	return Effect.runPromise(hubItemByIdCache.invalidateAll);
}
