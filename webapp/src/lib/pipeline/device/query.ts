// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createCachedFetchLoad } from '$lib/utils/cached-fetch-load.js';
import { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { z, ZodError } from 'zod';

import { pb } from '@/pocketbase';

import type { DeviceRecord } from './types';

const deviceWireSchema = z.object({
	name: z.string(),
	path: z.string(),
	runner_id: z.string(),
	runner_name: z.string(),
	description: z.string().optional(),
	is_owned: z.boolean(),
	is_published: z.boolean(),
	is_online: z.boolean(),
	url: z.string().optional(),
	type: z.string().optional(),
	queue_length: z.number().optional()
});

const listResponseSchema = z.object({
	devices: z.array(deviceWireSchema)
});

function mapWireToRecord(wire: z.infer<typeof deviceWireSchema>): DeviceRecord {
	return {
		name: wire.name,
		path: wire.path,
		runnerId: wire.runner_id,
		runnerName: wire.runner_name,
		description: wire.description,
		isOwned: wire.is_owned,
		isPublished: wire.is_published,
		isOnline: wire.is_online,
		url: wire.url,
		type: wire.type,
		queueLength: wire.queue_length
	};
}

export function parseSelectorResponse(body: unknown): DeviceRecord[] {
	const parsed = listResponseSchema.parse(body);
	return parsed.devices.map(mapWireToRecord);
}

export function fetchRecords(
	options: { fetch?: typeof fetch } = {}
): Task.Task<DeviceRecord[], ClientResponseError | ZodError> {
	const { fetch: fetchFn = fetch } = options;

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() =>
			pb.send('/api/mobile-devices', {
				method: 'GET',
				fetch: fetchFn,
				requestKey: null
			})
	).andThen((response) => {
		try {
			return Task.resolve(parseSelectorResponse(response));
		} catch (error) {
			return Task.reject(error as ZodError);
		}
	});
}

/**
 * Shared device-list cache for enrich / path resolution.
 * Kept separate from {@link fetchRecords} so the live Catalog can still refresh.
 */
const mobileDevicesListCache = createCachedFetchLoad<'list', DeviceRecord[]>({
	capacity: 1,
	lookup: async (_key, fetchFn) => {
		const response = await pb.send('/api/mobile-devices', {
			method: 'GET',
			fetch: fetchFn,
			requestKey: null
		});
		return parseSelectorResponse(response);
	}
});

export async function getCachedDeviceRecords(
	options: { fetch?: typeof fetch } = {}
): Promise<DeviceRecord[]> {
	return mobileDevicesListCache.get('list', { fetch: options.fetch ?? fetch });
}

/** Resolve one device by path using the shared list cache. */
export async function findCachedDeviceByPath(
	path: string,
	options: { fetch?: typeof fetch } = {}
): Promise<DeviceRecord | undefined> {
	const devices = await getCachedDeviceRecords(options);
	return devices.find((device) => device.path === path);
}

/** Clears cached device lists. Intended for tests. */
export function invalidateMobileDevicesCache(): Promise<void> {
	return mobileDevicesListCache.invalidateAll();
}
