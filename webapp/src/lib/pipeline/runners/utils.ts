// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ClientResponseError } from 'pocketbase';

import * as Task from 'true-myth/task';
import { z } from 'zod';

import { pb } from '@/pocketbase';

//

const mobileRunnerListItemSchema = z.object({
	description: z.string().optional(),
	mine: z.boolean(),
	name: z.string(),
	online: z.boolean(),
	published: z.boolean(),
	runner_id: z.string()
});

const mobileRunnersListResponseSchema = z.object({
	runners: z.array(mobileRunnerListItemSchema)
});

export type MobileRunnerListItem = z.infer<typeof mobileRunnerListItemSchema>;
export type MobileRunnerReference = Pick<MobileRunnerListItem, 'runner_id'>;

export function runnerID(runner: MobileRunnerReference): string {
	return runner.runner_id;
}

export async function fetchAvailableRunners(): Promise<MobileRunnerListItem[]> {
	const response = await pb.send<unknown>('/api/mobile-runners?view=selector', {
		method: 'GET',
		requestKey: null
	});
	return mobileRunnersListResponseSchema.parse(response).runners;
}

export function fetchAvailableForOrganization() {
	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() => fetchAvailableRunners()
	);
}

let snapshot: Promise<MobileRunnerListItem[]> | undefined;
let snapshotExpiresAt = 0;

function fetchStatusSnapshot(): Promise<MobileRunnerListItem[]> {
	const now = Date.now();
	if (snapshot && now < snapshotExpiresAt) return snapshot;

	snapshotExpiresAt = now + 1_000;
	snapshot = fetchAvailableRunners().catch((error) => {
		snapshot = undefined;
		snapshotExpiresAt = 0;
		throw error;
	});
	return snapshot;
}

export async function checkOnlineStatus(runner: MobileRunnerReference): Promise<boolean> {
	try {
		const runners = await fetchStatusSnapshot();
		return runners.find((item) => item.runner_id === runner.runner_id)?.online ?? false;
	} catch {
		return false;
	}
}
