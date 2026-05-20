// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { z, ZodError } from 'zod';

import { pb } from '@/pocketbase';

import type { RunnerRecord } from './types';

const runnerWireSchema = z.object({
	name: z.string(),
	path: z.string(),
	description: z.string().optional(),
	is_owned: z.boolean(),
	is_published: z.boolean(),
	is_online: z.boolean(),
	url: z.string().optional(),
	type: z.string().optional(),
	queue_length: z.number().optional()
});

const listResponseSchema = z.object({
	runners: z.array(runnerWireSchema)
});

function mapWireToRecord(wire: z.infer<typeof runnerWireSchema>): RunnerRecord {
	return {
		name: wire.name,
		path: wire.path,
		description: wire.description,
		isOwned: wire.is_owned,
		isPublished: wire.is_published,
		isOnline: wire.is_online,
		url: wire.url,
		type: wire.type,
		queueLength: wire.queue_length
	};
}

export function parseSelectorResponse(body: unknown): RunnerRecord[] {
	const parsed = listResponseSchema.parse(body);
	return parsed.runners.map(mapWireToRecord);
}

export function listSelector(
	options: { fetch?: typeof fetch } = {}
): Task.Task<RunnerRecord[], ClientResponseError | ZodError> {
	const { fetch: fetchFn = fetch } = options;

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() =>
			pb.send('/api/mobile-runners?view=selector', {
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
