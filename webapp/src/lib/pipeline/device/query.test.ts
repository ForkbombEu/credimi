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

import {
	getCachedDeviceRecords,
	invalidateMobileDevicesCache,
	parseSelectorResponse,
	resolveByPath
} from './query';

const wireDevice = {
	name: 'Online owned',
	path: 'usera-s-organization/owned-host/device-a',
	runner_id: 'usera-s-organization/owned-host',
	runner_name: 'Owned host',
	description: 'desc',
	is_owned: true,
	is_published: false,
	is_online: true
};

describe('parseSelectorResponse', () => {
	it('maps snake_case API body to DeviceRecord', () => {
		const records = parseSelectorResponse({
			devices: [wireDevice]
		});

		expect(records).toEqual([
			{
				name: 'Online owned',
				path: 'usera-s-organization/owned-host/device-a',
				runnerId: 'usera-s-organization/owned-host',
				runnerName: 'Owned host',
				description: 'desc',
				isOwned: true,
				isPublished: false,
				isOnline: true
			}
		]);
	});
});

describe('resolveByPath sticky device list', () => {
	beforeEach(async () => {
		send.mockReset();
		await invalidateMobileDevicesCache();
	});

	it('dedupes concurrent list loads into one network call', async () => {
		let resolveSend!: (value: { devices: (typeof wireDevice)[] }) => void;
		send.mockImplementation(
			() =>
				new Promise((resolve) => {
					resolveSend = resolve;
				})
		);

		const pending = Promise.all([
			getCachedDeviceRecords(),
			getCachedDeviceRecords(),
			resolveByPath(wireDevice.path)
		]);

		await vi.waitFor(() => expect(send).toHaveBeenCalledTimes(1));

		resolveSend({ devices: [wireDevice] });
		const [listA, listB, found] = await pending;

		expect(listA).toHaveLength(1);
		expect(listB).toEqual(listA);
		expect(found?.path).toBe(wireDevice.path);
		expect(send).toHaveBeenCalledTimes(1);
	});

	it('reuses a successful list on later lookups', async () => {
		send.mockResolvedValue({ devices: [wireDevice] });

		await getCachedDeviceRecords();
		await expect(resolveByPath(wireDevice.path)).resolves.toMatchObject({
			path: wireDevice.path,
			name: 'Online owned'
		});
		await expect(resolveByPath('missing/path')).resolves.toBeUndefined();

		expect(send).toHaveBeenCalledTimes(1);
	});
});
