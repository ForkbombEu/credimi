// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { checkAuthFlagAndUser, getUserOrganization } from '$lib/utils';
import { fetchWorkflowExecution, fetchWorkflowHistory, getWorkflowMemo } from '$lib/workflows';

//

export const load = async ({ params, fetch }) => {
	await checkAuthFlagAndUser({ fetch });

	const organization = await getUserOrganization({ fetch });
	if (!organization) {
		error(403, { message: 'You are not authorized to access this page' });
	}

	const workflow = await _getWorkflow(params.workflow_id, params.run_id, { fetch });
	if (workflow instanceof Error) {
		error(500, { message: workflow.message });
	}

	return {
		organization,
		workflow
	};
};

//

export async function _getWorkflow(workflowId: string, runId: string, options = { fetch }) {
	const fetched = await fetchWorkflowExecution(workflowId, runId, options);
	if (fetched instanceof Error) return fetched;

	const eventHistory = await fetchWorkflowHistory(workflowId, runId, options);
	if (eventHistory instanceof Error) return eventHistory;

	const memo = getWorkflowMemo(fetched.execution);
	if (memo instanceof Error) return memo;

	return {
		execution: fetched.execution,
		eventHistory,
		memo,
		devices: fetched.devices ?? []
	};
}
