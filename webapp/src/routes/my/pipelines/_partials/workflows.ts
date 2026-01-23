// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';

import { pb } from '@/pocketbase';

//

const baseUrl = '/api/pipeline/list-workflows';

export type PipelinesWorkflows = Record<string, WorkflowExecutionSummary[]>;

export async function getAllPipelinesWorkflows(options = { fetch }): Promise<PipelinesWorkflows> {
	const response = await pb.send(baseUrl, {
		method: 'GET',
		fetch: options.fetch
	});

	return response as PipelinesWorkflows;
}

export async function getPipelineWorkflows(
	pipelineId: string,
	options = { fetch }
): Promise<WorkflowExecutionSummary[]> {
	const response = await pb.send(`${baseUrl}/${pipelineId}`, {
		method: 'GET',
		fetch: options.fetch
	});
	return response as WorkflowExecutionSummary[];
}
