// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { checkAuthFlagAndUser, getUserOrganization } from '$lib/utils';
import { fetchWorkflowExecution, getWorkflowMemo } from '$lib/workflows';
import { ClientResponseError } from 'pocketbase';

import { m, redirect } from '@/i18n';

//

export const load = async ({ params, fetch }) => {
	await checkAuthFlagAndUser({ fetch });

	const organization = await getUserOrganization({ fetch });
	if (!organization) {
		error(403, { message: 'You are not authorized to access this page' });
	}

	const workflow = await _getWorkflow(params.workflow_id, params.run_id, { fetch });
	if (workflow instanceof ClientResponseError) {
		if (workflow.status === 401) redirect('/login');
		// Another organization's run, e.g. opened from a notification or link
		// meant for another account.
		if (workflow.status === 403) error(403, { message: m.Run_belongs_to_another_account() });
		if (workflow.status === 404) error(404);
	}
	if (workflow instanceof Error) {
		error(500, { message: workflow.message });
	}

	return {
		organization,
		workflow
	};
};

//

/** Describes the run only; the page fetches the event history (tens of MB for large pipelines) incrementally. */
export async function _getWorkflow(workflowId: string, runId: string, options = { fetch }) {
	const fetched = await fetchWorkflowExecution(workflowId, runId, options);
	if (fetched instanceof Error) return fetched;

	const memo = getWorkflowMemo(fetched.info.memo);
	if (memo instanceof Error) return memo;

	return {
		info: fetched.info,
		status: fetched.status,
		failure_reason: fetched.failure_reason,
		memo,
		devices: fetched.devices
	};
}
