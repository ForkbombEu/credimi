// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { fetchUserWorkflows } from '$lib/workflows/index.js';

import { error } from '@sveltejs/kit';

//

export const load = async ({ fetch }) => {
	const workflows = await fetchUserWorkflows(fetch);
	if (!workflows.success) {
		error(500, {
			message: 'Failed to parse response'
		});
	}

	return {
		executions: workflows.data.executions
	};
};
