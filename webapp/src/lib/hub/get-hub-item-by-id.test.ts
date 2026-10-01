// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

const getOne = vi.fn();

vi.mock('@/pocketbase', () => ({
	pb: {
		collection: () => ({
			getOne: (...args: unknown[]) => getOne(...args)
		})
	}
}));

import { getHubItemById, invalidateHubItemByIdCache } from './get-hub-item-by-id.js';

describe('getHubItemById', () => {
	beforeEach(async () => {
		getOne.mockReset();
		await invalidateHubItemByIdCache();
	});

	it('dedupes concurrent lookups for the same id into one network call', async () => {
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

	it('looks up distinct ids separately', async () => {
		getOne
			.mockResolvedValueOnce({ id: 'a', name: 'A' })
			.mockResolvedValueOnce({ id: 'b', name: 'B' });

		const [left, right] = await Promise.all([getHubItemById('a'), getHubItemById('b')]);

		expect(left).toEqual({ id: 'a', name: 'A' });
		expect(right).toEqual({ id: 'b', name: 'B' });
		expect(getOne).toHaveBeenCalledTimes(2);
	});
});
