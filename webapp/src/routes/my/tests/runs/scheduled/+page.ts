// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { loadScheduledWorkflows } from '$lib/workflows/schedule';

//

export const load = async ({ fetch }) => {
	const res = await loadScheduledWorkflows({ fetch });
	return {
		scheduledWorkflows: res
	};
};
