// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { runWithLoading } from '$lib/utils';

import { pb } from '@/pocketbase';

//

export function cancelWorkflow(execution: { workflowId: string; runId: string }) {
	const { workflowId, runId } = execution;
	runWithLoading({
		fn: async () => {
			await pb.send(`/api/my/checks/${workflowId}/runs/${runId}/terminate`, {
				method: 'POST'
			});
		}
	});
}
