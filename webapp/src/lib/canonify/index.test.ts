// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

const send = vi.fn();

vi.mock('@/pocketbase', () => ({
	pb: {
		send: (...args: unknown[]) => send(...args)
	}
}));

import { getRecordByCanonifiedPath, invalidateCanonifyPathCache } from './index.js';

describe('getRecordByCanonifiedPath', () => {
	beforeEach(async () => {
		send.mockReset();
		await invalidateCanonifyPathCache();
	});

	it('dedupes concurrent lookups for the same path into one network call', async () => {
		let resolveSend!: (value: { message: string; record: { id: string } }) => void;
		send.mockImplementation(
			() =>
				new Promise((resolve) => {
					resolveSend = resolve;
				})
		);

		const pending = Promise.all([
			getRecordByCanonifiedPath('org/wallet/action-a'),
			getRecordByCanonifiedPath('org/wallet/action-a'),
			getRecordByCanonifiedPath('org/wallet/action-a')
		]);

		await vi.waitFor(() => expect(send).toHaveBeenCalledTimes(1));

		resolveSend({ message: 'valid identifier', record: { id: 'rec1' } });
		const results = await pending;

		expect(results).toEqual([{ id: 'rec1' }, { id: 'rec1' }, { id: 'rec1' }]);
		expect(send).toHaveBeenCalledTimes(1);
	});

	it('reuses a successful lookup on later calls', async () => {
		send.mockResolvedValue({ message: 'valid identifier', record: { id: 'rec1' } });

		await expect(getRecordByCanonifiedPath('org/wallet/action-b')).resolves.toEqual({
			id: 'rec1'
		});
		await expect(getRecordByCanonifiedPath('org/wallet/action-b')).resolves.toEqual({
			id: 'rec1'
		});

		expect(send).toHaveBeenCalledTimes(1);
	});

	it('does not sticky-cache failures so a later call can succeed', async () => {
		send.mockRejectedValueOnce(new Error('429'));
		send.mockResolvedValueOnce({ message: 'valid identifier', record: { id: 'rec2' } });

		await expect(getRecordByCanonifiedPath('org/wallet/action-c')).resolves.toEqual(
			new Error('Failed to get record by path')
		);
		await expect(getRecordByCanonifiedPath('org/wallet/action-c')).resolves.toEqual({
			id: 'rec2'
		});

		expect(send).toHaveBeenCalledTimes(2);
	});

	it('looks up distinct paths separately', async () => {
		send.mockResolvedValueOnce({
			message: 'valid identifier',
			record: { id: 'a' }
		}).mockResolvedValueOnce({ message: 'valid identifier', record: { id: 'b' } });

		const [left, right] = await Promise.all([
			getRecordByCanonifiedPath('org/wallet/action-d'),
			getRecordByCanonifiedPath('org/wallet/action-e')
		]);

		expect(left).toEqual({ id: 'a' });
		expect(right).toEqual({ id: 'b' });
		expect(send).toHaveBeenCalledTimes(2);
	});
});
