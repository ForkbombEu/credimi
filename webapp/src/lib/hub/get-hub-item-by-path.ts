// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Cache, Duration, Effect, Exit, FiberRef, GlobalValue } from 'effect';

import { pb } from '@/pocketbase';

import type { HubItem } from './types.js';

/**
 * Per-fiber fetch override so SSR can pass SvelteKit's `fetch` while the
 * shared path cache still dedupes concurrent lookups for the same path.
 *
 * @see https://effect.website/docs/caching/cache/
 */
const hubItemByPathFetchRef = GlobalValue.globalValue(
	Symbol.for('@credimi/hub-item-by-path/fetch'),
	() => FiberRef.unsafeMake<typeof fetch>(fetch)
);

/**
 * Path → hub_items record cache. Concurrent `get`s for the same path share one
 * lookup. Failures use TTL zero so transient errors (e.g. 429) are not sticky.
 */
const hubItemByPathCache = Effect.runSync(
	Cache.makeWith<string, HubItem, Error>({
		capacity: 2048,
		lookup: (path) =>
			Effect.gen(function* () {
				const fetchFn = yield* FiberRef.get(hubItemByPathFetchRef);
				return yield* Effect.tryPromise({
					try: () =>
						pb
							.collection('hub_items')
							.getFirstListItem<HubItem>(pb.filter('path ~ {:path}', { path }), {
								fetch: fetchFn,
								requestKey: null
							}),
					catch: (cause) =>
						cause instanceof Error
							? cause
							: new Error('Failed to get hub item by path')
				});
			}),
		timeToLive: (exit) => (Exit.isSuccess(exit) ? Duration.infinity : Duration.zero)
	})
);

export async function getHubItemByPath(path: string, options = { fetch }): Promise<HubItem> {
	return Effect.runPromise(
		hubItemByPathCache.get(path).pipe(
			Effect.catchAll((error) =>
				hubItemByPathCache.invalidate(path).pipe(Effect.andThen(Effect.fail(error)))
			),
			Effect.locally(hubItemByPathFetchRef, options.fetch)
		)
	);
}

/** Clears cached hub_items path lookups. Intended for tests. */
export function invalidateHubItemByPathCache(): Promise<void> {
	return Effect.runPromise(hubItemByPathCache.invalidateAll);
}
