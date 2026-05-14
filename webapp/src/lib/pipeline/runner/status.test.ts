// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { MobileRunnersResponse } from '@/pocketbase/types';

import { POLL_INTERVAL_MS, StatusCoordinator } from './status-coordinator';

//

function runner(id: string, path: string): MobileRunnersResponse {
	return {
		id,
		__canonified_path__: path,
		canonified_name: path.split('/').at(-1) ?? id,
		collectionId: 'mobile_runners',
		collectionName: 'mobile_runners',
		created: '2025-01-01',
		description: '',
		expand: {},
		ip: '127.0.0.1',
		name: `Runner ${id}`,
		owner: 'org1',
		port: 8080,
		published: true,
		updated: '2025-01-01'
	} as MobileRunnersResponse;
}

function createCoordinator(checkOnlineStatus: (runner: MobileRunnersResponse) => Promise<boolean>) {
	const updates: Array<{ online: boolean; path: string }> = [];
	let cleared = false;

	const coordinator = new StatusCoordinator(checkOnlineStatus, {
		onClear: () => {
			cleared = true;
		},
		onUpdate: (path, online) => {
			updates.push({ path, online });
		}
	});

	return { cleared: () => cleared, coordinator, updates };
}

describe('StatusCoordinator', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('dedupes runners by path before probing', async () => {
		const check = vi.fn(async () => true);
		const { coordinator, updates } = createCoordinator(check);
		const r = runner('1', '/org/runner-a');

		coordinator.probe([r, r, r], { reason: 'periodic' });
		await Promise.resolve();

		expect(check).toHaveBeenCalledTimes(1);
		expect(updates).toEqual([{ path: '/org/runner-a', online: true }]);
	});

	it('ignores slower probe results superseded by a newer probe for the same path', async () => {
		const resolvers: Array<(value: boolean) => void> = [];
		const check = vi.fn(
			() =>
				new Promise<boolean>((resolve) => {
					resolvers.push(resolve);
				})
		);
		const { coordinator, updates } = createCoordinator(check);
		const r = runner('1', '/org/runner-a');

		coordinator.probe([r], { reason: 'periodic' });
		coordinator.probe([r], { reason: 'periodic' });

		resolvers[0]?.(true);
		await Promise.resolve();
		expect(updates).toHaveLength(0);

		resolvers[1]?.(false);
		await Promise.resolve();
		expect(updates).toEqual([{ path: '/org/runner-a', online: false }]);
	});

	it('ignores completions from a previous modal burst after a new modal open', async () => {
		const resolvers: Array<(value: boolean) => void> = [];
		const check = vi.fn(
			() =>
				new Promise<boolean>((resolve) => {
					resolvers.push(resolve);
				})
		);
		const { coordinator, updates } = createCoordinator(check);
		const r = runner('1', '/org/runner-a');

		coordinator.probe([r], { reason: 'modal' });
		coordinator.probe([r], { reason: 'modal' });

		resolvers[0]?.(true);
		await Promise.resolve();
		expect(updates).toHaveLength(0);

		resolvers[1]?.(false);
		await Promise.resolve();
		expect(updates).toEqual([{ path: '/org/runner-a', online: false }]);
	});

	it('startPolling is idempotent and polls on the configured interval', async () => {
		const check = vi.fn(async () => true);
		const { coordinator } = createCoordinator(check);
		const runners = [runner('1', '/org/runner-a')];
		const getRunners = vi.fn(() => runners);

		coordinator.startPolling(getRunners);
		coordinator.startPolling(getRunners);
		await Promise.resolve();

		expect(getRunners).toHaveBeenCalledTimes(1);
		expect(check).toHaveBeenCalledTimes(1);

		vi.advanceTimersByTime(POLL_INTERVAL_MS);
		await Promise.resolve();

		expect(getRunners).toHaveBeenCalledTimes(2);
		expect(check).toHaveBeenCalledTimes(2);
	});

	it('stopPolling clears state and prevents further interval probes', async () => {
		const check = vi.fn(async () => true);
		const { cleared, coordinator, updates } = createCoordinator(check);
		const runners = [runner('1', '/org/runner-a')];

		coordinator.startPolling(() => runners);
		await Promise.resolve();
		expect(updates).toHaveLength(1);

		coordinator.stopPolling();
		expect(cleared()).toBe(true);
		expect(coordinator.read()).toEqual({});

		updates.length = 0;
		vi.advanceTimersByTime(POLL_INTERVAL_MS);
		await Promise.resolve();

		expect(check).toHaveBeenCalledTimes(1);
		expect(updates).toHaveLength(0);
	});
});
