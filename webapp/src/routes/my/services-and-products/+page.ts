// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { fetchWorkflows } from '$lib/workflows/index.js';
import { error } from '@sveltejs/kit';

//

export const load = async ({ fetch }) => {
	const workflows = await fetchWorkflows({ fetch });
	if (workflows instanceof Error) {
		error(500, { message: workflows.message });
	}

	return {
		workflows
	};
};
