// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Cache, Duration, Effect, Exit, FiberRef, GlobalValue } from 'effect';

/**
 * Shared per-fiber fetch override for cached loaders (canonify, hub, …).
 * SSR can pass SvelteKit's `fetch` via {@link CachedFetchLoad.get}; concurrent
 * lookups still dedupe across callers that share the same cache key.
 *
 * @see https://effect.website/docs/caching/cache/
 */
export const requestFetchRef = GlobalValue.globalValue(Symbol.for('@credimi/request-fetch'), () =>
	FiberRef.unsafeMake<typeof fetch>(fetch)
);

export type CachedFetchLoadOptions<Key, Value> = {
	/** Max entries retained. Defaults to 2048. */
	capacity?: number;
	/**
	 * Perform the network/domain lookup. Receives the active fiber's `fetch`
	 * (SSR override or global). Throw to mark the lookup as failed.
	 */
	lookup: (key: Key, fetch: typeof globalThis.fetch) => Promise<Value>;
};

export type CachedFetchLoad<Key, Value> = {
	/** Resolve a value; throws on failure after dropping the failed cache entry. */
	get: (key: Key, options?: { fetch: typeof globalThis.fetch }) => Promise<Value>;
	/**
	 * Like {@link CachedFetchLoad.get}, but returns the `Error` instead of
	 * throwing (used by soft-failure callers such as canonify).
	 */
	getOrError: (key: Key, options?: { fetch: typeof globalThis.fetch }) => Promise<Value | Error>;
	/** Clears all entries. Intended for tests. */
	invalidateAll: () => Promise<void>;
};

const DEFAULT_CAPACITY = 2048;

/**
 * Build a string-keyed (or otherwise hashable) Effect Cache with sticky successes,
 * non-sticky failures, concurrent-get dedupe, and shared SSR `fetch` via
 * {@link requestFetchRef}.
 */
export function createCachedFetchLoad<Key, Value>(
	options: CachedFetchLoadOptions<Key, Value>
): CachedFetchLoad<Key, Value> {
	const cache = Effect.runSync(
		Cache.makeWith<Key, Value, Error>({
			capacity: options.capacity ?? DEFAULT_CAPACITY,
			lookup: (key) =>
				Effect.gen(function* () {
					const fetchFn = yield* FiberRef.get(requestFetchRef);
					return yield* Effect.tryPromise({
						try: () => options.lookup(key, fetchFn),
						catch: (cause) =>
							cause instanceof Error ? cause : new Error(String(cause))
					});
				}),
			timeToLive: (exit) => (Exit.isSuccess(exit) ? Duration.infinity : Duration.zero)
		})
	);

	function withFetch<A, E>(
		key: Key,
		fetchFn: typeof fetch,
		onError: (error: Error) => Effect.Effect<A, E>
	): Effect.Effect<A | Value, E> {
		return cache.get(key).pipe(
			Effect.catchAll((error) =>
				// Drop failed entries so transient errors (e.g. 429) can be retried.
				cache.invalidate(key).pipe(Effect.andThen(onError(error)))
			),
			Effect.locally(requestFetchRef, fetchFn)
		);
	}

	return {
		get(key, opts = { fetch }) {
			return Effect.runPromise(withFetch(key, opts.fetch, (error) => Effect.fail(error)));
		},
		getOrError(key, opts = { fetch }) {
			return Effect.runPromise(withFetch(key, opts.fetch, (error) => Effect.succeed(error)));
		},
		invalidateAll() {
			return Effect.runPromise(cache.invalidateAll);
		}
	};
}
