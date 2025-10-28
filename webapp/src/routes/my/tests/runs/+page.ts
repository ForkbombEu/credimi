// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { isWorkflowStatus, type WorkflowStatusType } from '$lib/temporal';
import { fetchWorkflows, WORKFLOW_STATUS_QUERY_PARAM } from '$lib/workflows/index.js';

import { redirect } from '@/i18n/index.js';

//

export const load = async ({ fetch, url }) => {
	const status = url.searchParams.get(WORKFLOW_STATUS_QUERY_PARAM);

	let parsedStatus: WorkflowStatusType | undefined = undefined;
	if (status) {
		if (isWorkflowStatus(status)) {
			parsedStatus = status;
		} else {
			redirect('/my/tests/runs');
		}
	}

	const workflows = await fetchWorkflows({ fetch, status: parsedStatus });
	if (workflows instanceof Error) {
		error(500, {
			message: workflows.message
		});
	}

	return {
		workflows: workflows.executions,
		selectedStatus: parsedStatus
	};
};
