// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';

import { pb } from '@/pocketbase';

//

const baseUrl = '/api/pipeline/list-workflows';

export async function getAllPipelinesWorkflows(options = { fetch }) {
	return pb.send<Record<string, WorkflowExecutionSummary[]>>(baseUrl, {
		method: 'GET',
		fetch: options.fetch
	});
}

export async function getPipelineWorkflows(pipelineId: string, options = { fetch }) {
	return pb.send<WorkflowExecutionSummary[]>(`${baseUrl}/${pipelineId}`, {
		method: 'GET',
		fetch: options.fetch
	});
}
