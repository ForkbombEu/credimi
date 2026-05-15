// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ClientResponseError } from 'pocketbase';

import * as Task from 'true-myth/task';
import { z } from 'zod';

import type { MobileRunnersResponse } from '@/pocketbase/types';

import { pb } from '@/pocketbase';

//

export function fetchAvailableForOrganization(organizationId: string) {
	const filter = pb.filter('owner.id = {:currentOrganization} || published = true', {
		currentOrganization: organizationId
	});
	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() =>
			pb.collection('mobile_runners').getFullList({
				requestKey: null,
				filter: filter
			})
	);
}

function getHealthUrl(runner: MobileRunnersResponse): string {
	let baseUrl = runner.ip;
	if (runner.port) baseUrl = `${baseUrl}:${runner.port}`;
	return `${baseUrl}/health`;
}

const isOnlineResponseSchema = z.object({
	status: z.literal('connected')
});

export async function checkOnlineStatus(runner: MobileRunnersResponse): Promise<boolean> {
	try {
		const response = await fetch(getHealthUrl(runner), { method: 'GET' });
		if (!response.ok) return false;
		return isOnlineResponseSchema.safeParse(await response.json()).success;
	} catch {
		return false;
	}
}
