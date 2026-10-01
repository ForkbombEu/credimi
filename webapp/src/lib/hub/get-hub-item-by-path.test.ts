// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

const getFirstListItem = vi.fn();
const filter = vi.fn((expr: string) => expr);

vi.mock('@/pocketbase', () => ({
	pb: {
		filter: (...args: unknown[]) => filter(...args),
		collection: () => ({
			getFirstListItem: (...args: unknown[]) => getFirstListItem(...args)
		})
	}
}));

import { getHubItemByPath, invalidateHubItemByPathCache } from './get-hub-item-by-path.js';

describe('getHubItemByPath', () => {
	beforeEach(async () => {
		getFirstListItem.mockReset();
		filter.mockClear();
		await invalidateHubItemByPathCache();
	});

	it('dedupes concurrent lookups for the same path into one network call', async () => {
		let resolveGet!: (value: { id: string; path: string }) => void;
		getFirstListItem.mockImplementation(
			() =>
				new Promise((resolve) => {
					resolveGet = resolve;
				})
		);

		const pending = Promise.all([
			getHubItemByPath('/org/cred-a'),
			getHubItemByPath('/org/cred-a'),
			getHubItemByPath('/org/cred-a')
		]);

		await vi.waitFor(() => expect(getFirstListItem).toHaveBeenCalledTimes(1));

		resolveGet({ id: 'cred-1', path: '/org/cred-a' });
		const results = await pending;

		expect(results).toEqual([
			{ id: 'cred-1', path: '/org/cred-a' },
			{ id: 'cred-1', path: '/org/cred-a' },
			{ id: 'cred-1', path: '/org/cred-a' }
		]);
		expect(getFirstListItem).toHaveBeenCalledTimes(1);
	});

	it('reuses a successful lookup on later calls', async () => {
		getFirstListItem.mockResolvedValue({ id: 'cred-2', path: '/org/cred-b' });

		await expect(getHubItemByPath('/org/cred-b')).resolves.toEqual({
			id: 'cred-2',
			path: '/org/cred-b'
		});
		await expect(getHubItemByPath('/org/cred-b')).resolves.toEqual({
			id: 'cred-2',
			path: '/org/cred-b'
		});

		expect(getFirstListItem).toHaveBeenCalledTimes(1);
	});
});
