// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { isWorkflowStatus, type WorkflowStatusType } from '$lib/temporal';
import { fetchWorkflows, groupWorkflowsWithChildren } from '$lib/workflows/index.js';

import { redirect } from '@/i18n/index.js';

//

export const load = async ({ fetch, url }) => {
	const status = url.searchParams.get('status');
	if (status && !isWorkflowStatus(status)) {
		redirect('/my/tests/runs');
	}

	const workflows = await fetchWorkflows({ fetch, statuses: [status as WorkflowStatusType] });
	if (workflows instanceof Error) {
		error(500, {
			message: workflows.message
		});
	}

	return {
		workflows: groupWorkflowsWithChildren(workflows)
	};
};
