// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

const getOne = vi.fn();
const getFirstListItem = vi.fn();
const filter = vi.fn((expr: string) => expr);

vi.mock('@/pocketbase', () => ({
	pb: {
		filter: (...args: unknown[]) => filter(...args),
		collection: () => ({
			getOne: (...args: unknown[]) => getOne(...args),
			getFirstListItem: (...args: unknown[]) => getFirstListItem(...args)
		})
	}
}));

import {
	getHubItemById,
	getHubItemByPath,
	invalidateHubItemByIdCache,
	invalidateHubItemByPathCache
} from './get-hub-item.js';

describe('getHubItem cache (shared)', () => {
	beforeEach(async () => {
		getOne.mockReset();
		getFirstListItem.mockReset();
		filter.mockClear();
		await invalidateHubItemByIdCache();
		await invalidateHubItemByPathCache();
	});

	it('dedupes concurrent lookups for the same key into one network call', async () => {
		let resolveGet!: (value: { id: string; name: string }) => void;
		getOne.mockImplementation(
			() =>
				new Promise((resolve) => {
					resolveGet = resolve;
				})
		);

		const pending = Promise.all([
			getHubItemById('wallet-1'),
			getHubItemById('wallet-1'),
			getHubItemById('wallet-1')
		]);

		await vi.waitFor(() => expect(getOne).toHaveBeenCalledTimes(1));

		resolveGet({ id: 'wallet-1', name: 'Demo Wallet' });
		const results = await pending;

		expect(results).toEqual([
			{ id: 'wallet-1', name: 'Demo Wallet' },
			{ id: 'wallet-1', name: 'Demo Wallet' },
			{ id: 'wallet-1', name: 'Demo Wallet' }
		]);
		expect(getOne).toHaveBeenCalledTimes(1);
	});

	it('reuses a successful lookup on later calls', async () => {
		getOne.mockResolvedValue({ id: 'wallet-2', name: 'Cached Wallet' });

		await expect(getHubItemById('wallet-2')).resolves.toEqual({
			id: 'wallet-2',
			name: 'Cached Wallet'
		});
		await expect(getHubItemById('wallet-2')).resolves.toEqual({
			id: 'wallet-2',
			name: 'Cached Wallet'
		});

		expect(getOne).toHaveBeenCalledTimes(1);
	});

	it('does not sticky-cache failures so a later call can succeed', async () => {
		getOne.mockRejectedValueOnce(new Error('429'));
		getOne.mockResolvedValueOnce({ id: 'wallet-3', name: 'Recovered Wallet' });

		await expect(getHubItemById('wallet-3')).rejects.toThrow('429');
		await expect(getHubItemById('wallet-3')).resolves.toEqual({
			id: 'wallet-3',
			name: 'Recovered Wallet'
		});

		expect(getOne).toHaveBeenCalledTimes(2);
	});

	it('looks up distinct keys separately', async () => {
		getOne
			.mockResolvedValueOnce({ id: 'a', name: 'A' })
			.mockResolvedValueOnce({ id: 'b', name: 'B' });

		const [left, right] = await Promise.all([getHubItemById('a'), getHubItemById('b')]);

		expect(left).toEqual({ id: 'a', name: 'A' });
		expect(right).toEqual({ id: 'b', name: 'B' });
		expect(getOne).toHaveBeenCalledTimes(2);
	});
});

describe('getHubItemById lookup', () => {
	beforeEach(async () => {
		getOne.mockReset();
		getFirstListItem.mockReset();
		await invalidateHubItemByIdCache();
	});

	it('resolves via collection getOne', async () => {
		getOne.mockResolvedValue({ id: 'wallet-1', name: 'Demo Wallet' });

		await expect(getHubItemById('wallet-1')).resolves.toEqual({
			id: 'wallet-1',
			name: 'Demo Wallet'
		});

		expect(getOne).toHaveBeenCalledWith('wallet-1', {
			fetch: expect.any(Function),
			requestKey: null
		});
		expect(getFirstListItem).not.toHaveBeenCalled();
	});
});

describe('getHubItemByPath lookup', () => {
	beforeEach(async () => {
		getOne.mockReset();
		getFirstListItem.mockReset();
		filter.mockClear();
		await invalidateHubItemByPathCache();
	});

	it('resolves via getFirstListItem with path filter', async () => {
		getFirstListItem.mockResolvedValue({ id: 'cred-1', path: '/org/cred-a' });

		await expect(getHubItemByPath('/org/cred-a')).resolves.toEqual({
			id: 'cred-1',
			path: '/org/cred-a'
		});

		expect(filter).toHaveBeenCalledWith('path ~ {:path}', { path: '/org/cred-a' });
		expect(getFirstListItem).toHaveBeenCalledWith('path ~ {:path}', {
			fetch: expect.any(Function),
			requestKey: null
		});
		expect(getOne).not.toHaveBeenCalled();
	});
});
