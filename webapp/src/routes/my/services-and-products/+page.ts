// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { fetchUserWorkflows } from '$lib/workflows/index.js';

//

export const load = async ({ fetch }) => {
	const workflows = await fetchUserWorkflows(fetch);

	return {
		workflows: workflows.data?.executions
	};
};
