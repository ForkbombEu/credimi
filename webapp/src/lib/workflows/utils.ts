// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { workflowsResponseSchema } from './types';

export async function fetchUserWorkflows(fetchFn = fetch) {
	const data = await pb.send('/api/compliance/checks', {
		method: 'GET',
		fetch: fetchFn
	});
	return workflowsResponseSchema.safeParse(data);
}
