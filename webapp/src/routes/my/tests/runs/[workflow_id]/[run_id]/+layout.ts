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

	const data = await _loadData(workflow_id, run_id, { fetch });
	if (data instanceof Error) {
		error(500, { message: data.message });
	}

	//

	return {
		workflowId: workflow_id,
		runId: run_id,
		organization,
		...data
	};
};

//

export async function _loadData(workflowId: string, runId: string, options = { fetch }) {
	const basePath = `/api/compliance/checks/${workflowId}/${runId}`;

	//

	const workflowResponse = await pb.send(basePath, {
		method: 'GET',
		fetch: options.fetch
	});
	const workflowResponseValidation = rawWorkflowResponseSchema.safeParse(workflowResponse);
	if (!workflowResponseValidation.success) {
		return workflowResponseValidation.error;
	}

	//

	const historyResponse = await pb.send(`${basePath}/history`, {
		method: 'GET',
		fetch: options.fetch
	});
	const historyResponseValidation = rawHistoryResponseSchema.safeParse(historyResponse);
	if (!historyResponseValidation.success) {
		return historyResponseValidation.error;
	}

	return {
		workflow: workflowResponseValidation.data,
		workflowMemo: getWorkflowMemo(workflowResponseValidation.data.workflowExecutionInfo),
		eventHistory: historyResponseValidation.data as HistoryEvent[]
	};
}

const rawWorkflowResponseSchema = z
	.object({
		workflowExecutionInfo: workflowExecutionSchema
	})
	.passthrough();

export type WorkflowResponse = z.infer<typeof rawWorkflowResponseSchema>;

const rawHistoryResponseSchema = z.array(z.record(z.unknown()));
