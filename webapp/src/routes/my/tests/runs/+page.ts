// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { fetchWorkflows } from '$lib/workflows/index.js';

import { error } from '@sveltejs/kit';
import { getWorkflowStatusesFromUrl, INVALIDATE_KEY } from './utils';

//

export const load = async ({ fetch, url, depends }) => {
	depends(INVALIDATE_KEY);

	const statuses = getWorkflowStatusesFromUrl(url);

	const workflows = await fetchWorkflows({ fetch, statuses });
	if (workflows instanceof Error) {
		error(500, {
			message: workflows.message
		});
	}

	return {
		workflows,
		selectedStatuses: statuses
	};
};
