// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

import { createCachedFetchLoad } from './cached-fetch-load.js';

describe('createCachedFetchLoad', () => {
	const lookup = vi.fn();
	let cache: ReturnType<typeof createCachedFetchLoad<string, { id: string }>>;

	beforeEach(async () => {
		lookup.mockReset();
		cache = createCachedFetchLoad({ lookup });
		await cache.invalidateAll();
	});

	it('dedupes concurrent gets for the same key into one lookup', async () => {
		let resolveLookup!: (value: { id: string }) => void;
		lookup.mockImplementation(
			() =>
				new Promise((resolve) => {
					resolveLookup = resolve;
				})
		);

		const pending = Promise.all([cache.get('a'), cache.get('a'), cache.get('a')]);

		await vi.waitFor(() => expect(lookup).toHaveBeenCalledTimes(1));

		resolveLookup({ id: 'a' });
		await expect(pending).resolves.toEqual([{ id: 'a' }, { id: 'a' }, { id: 'a' }]);
		expect(lookup).toHaveBeenCalledTimes(1);
	});

	it('reuses successful lookups', async () => {
		lookup.mockResolvedValue({ id: 'b' });

		await expect(cache.get('b')).resolves.toEqual({ id: 'b' });
		await expect(cache.get('b')).resolves.toEqual({ id: 'b' });

		expect(lookup).toHaveBeenCalledTimes(1);
	});

	it('does not sticky-cache failures', async () => {
		lookup.mockRejectedValueOnce(new Error('429'));
		lookup.mockResolvedValueOnce({ id: 'c' });

		await expect(cache.get('c')).rejects.toThrow('429');
		await expect(cache.get('c')).resolves.toEqual({ id: 'c' });

		expect(lookup).toHaveBeenCalledTimes(2);
	});

	it('getOrError returns the Error instead of throwing', async () => {
		lookup.mockRejectedValueOnce(new Error('boom'));

		await expect(cache.getOrError('d')).resolves.toEqual(new Error('boom'));
	});

	it('passes the options.fetch into lookup', async () => {
		const customFetch = vi.fn() as unknown as typeof fetch;
		lookup.mockResolvedValue({ id: 'e' });

		await cache.get('e', { fetch: customFetch });

		expect(lookup).toHaveBeenCalledWith('e', customFetch);
	});
});
