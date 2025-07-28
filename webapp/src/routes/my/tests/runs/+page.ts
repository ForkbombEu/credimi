// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { fetchWorkflows } from '$lib/workflows/index.js';

import { error } from '@sveltejs/kit';
import { getWorkflowStatusesFromUrl } from './utils';

//

export const load = async ({ fetch, url }) => {
	const statuses = getWorkflowStatusesFromUrl(url);

	const workflows = await fetchWorkflows({ fetch, statuses });
	if (workflows instanceof Error) {
		error(500, {
			message: workflows.message
		});
	}
	console.log(workflows);

	return {
		workflows,
		selectedStatuses: statuses
	};
};
