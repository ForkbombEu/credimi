// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { fetchWorkflows, groupWorkflowsWithChildren } from '$lib/workflows/index.js';

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

	return {
		workflows: groupWorkflowsWithChildren(workflows),
		selectedStatuses: statuses
	};
};
