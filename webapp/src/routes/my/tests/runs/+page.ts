// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { fetchUserWorkflows } from '$lib/workflows/index.js';

import { error } from '@sveltejs/kit';
import { getWorkflowStatusesFromUrl } from './utils';

//

export const load = async ({ fetch, url }) => {
	const statuses = getWorkflowStatusesFromUrl(url);

	const workflows = await fetchUserWorkflows({ fetch, statuses });
	if (!workflows.success) {
		error(500, {
			message: 'Failed to parse response'
		});
	}

	return {
		executions: workflows.data.executions,
		selectedStatuses: statuses
	};
};
