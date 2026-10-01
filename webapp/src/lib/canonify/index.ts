// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Cache, Duration, Effect, Exit, FiberRef, GlobalValue } from 'effect';

import { pb } from '@/pocketbase';

//

type GetRecordByCanonifiedPathResult<T = unknown> = { message: string; record?: T };

/**
 * Per-fiber fetch override so SSR can pass SvelteKit's `fetch` while the
 * shared path cache still dedupes concurrent lookups for the same path.
 *
 * @see https://effect.website/docs/caching/cache/
 */
const canonifyFetchRef = GlobalValue.globalValue(Symbol.for('@credimi/canonify/fetch'), () =>
	FiberRef.unsafeMake<typeof fetch>(fetch)
);

/**
 * Path → record cache. Concurrent `get`s for the same path share one lookup.
 * Failures use TTL zero so transient errors (e.g. 429) are not sticky.
 */
const pathRecordCache = Effect.runSync(
	Cache.makeWith<string, unknown, Error>({
		capacity: 2048,
		lookup: (path) =>
			Effect.gen(function* () {
				const fetchFn = yield* FiberRef.get(canonifyFetchRef);
				const result = yield* Effect.tryPromise({
					try: () =>
						pb.send<GetRecordByCanonifiedPathResult>(
							'/api/canonify/identifier/validate',
							{
								method: 'POST',
								body: { canonified_name: path },
								fetch: fetchFn,
								requestKey: null
							}
						),
					catch: () => new Error('Failed to get record by path')
				});
				if (result.record) {
					return result.record;
				}
				return yield* Effect.fail(new Error(result.message));
			}),
		timeToLive: (exit) => (Exit.isSuccess(exit) ? Duration.infinity : Duration.zero)
	})
);

export async function getRecordByCanonifiedPath<T = unknown>(
	path: string,
	options = { fetch }
): Promise<T | Error> {
	return Effect.runPromise(
		pathRecordCache.get(path).pipe(
			Effect.map((record) => record as T),
			Effect.catchAll((error) =>
				// Drop failed entries so transient errors (e.g. 429) can be retried.
				pathRecordCache.invalidate(path).pipe(Effect.as(error))
			),
			Effect.locally(canonifyFetchRef, options.fetch)
		)
	);
}

/** Clears cached path lookups. Intended for tests. */
export function invalidateCanonifyPathCache(): Promise<void> {
	return Effect.runPromise(pathRecordCache.invalidateAll);
}
