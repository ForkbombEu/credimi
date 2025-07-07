// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { workflowsResponseSchema } from './types';
import type { WorkflowStatusType } from '$lib/temporal';
import { String } from 'effect';

//

type Options = {
	fetch: typeof fetch;
	statuses: WorkflowStatusType[];
};

export async function fetchUserWorkflows(options: Partial<Options> = {}) {
	const { fetch: fetchFn = fetch, statuses = [] } = options;

	let url = '/api/compliance/checks';
	if (statuses.length > 0) {
		const formattedStatuses = statuses.map((status) => String.pascalToSnake(status));
		url += `?status=${formattedStatuses.join(',')}`;
	}

	const data = await pb.send(url, {
		method: 'GET',
		fetch: fetchFn
	});
	return workflowsResponseSchema.safeParse(data);
}
