// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import { z } from 'zod';
import { error } from '@sveltejs/kit';
import type { HistoryEvent } from '@forkbombeu/temporal-ui';
import { getWorkflowMemo, workflowExecutionSchema } from '$lib/workflows';
import { checkAuthFlagAndUser, getUserOrganization } from '$lib/utils';

//

export const load = async ({ params, fetch }) => {
	await checkAuthFlagAndUser({ fetch });
	const organization = await getUserOrganization({ fetch });

	const { workflow_id, run_id } = params;

	const basePath = `/api/compliance/checks/${workflow_id}/${run_id}`;

	//

	const workflowResponse = await pb.send(basePath, {
		method: 'GET',
		fetch
	});
	const workflowResponseValidation = rawWorkflowResponseSchema.safeParse(workflowResponse);
	if (!workflowResponseValidation.success) {
		error(500, { message: 'Failed to parse workflow response' });
	}

	//

	const historyResponse = await pb.send(`${basePath}/history`, {
		method: 'GET',
		fetch
	});
	const historyResponseValidation = rawHistoryResponseSchema.safeParse(historyResponse);
	if (!historyResponseValidation.success) {
		error(500, { message: 'Failed to parse workflow response' });
	}

	//

	return {
		workflowId: workflow_id,
		runId: run_id,
		eventHistory: historyResponseValidation.data as HistoryEvent[],
		workflow: workflowResponseValidation.data,
		workflowMemo: getWorkflowMemo(workflowResponseValidation.data.workflowExecutionInfo),
		organization
	};
};

//

const rawWorkflowResponseSchema = z
	.object({
		workflowExecutionInfo: workflowExecutionSchema
	})
	.passthrough();

export type WorkflowResponse = z.infer<typeof rawWorkflowResponseSchema>;

const rawHistoryResponseSchema = z.array(z.record(z.unknown()));
