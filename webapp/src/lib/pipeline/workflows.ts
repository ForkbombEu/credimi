// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Workflow } from '$lib';

import type { MobileRunnersResponse } from '@/pocketbase/types';

import { pb } from '@/pocketbase';

import StatusTag from './workflow-status-tag.svelte';
import SmallTable from './workflows-table-small.svelte';
import Table from './workflows-table.svelte';
export { SmallTable, StatusTag, Table };

//

export const QUEUED_STATUS = 'Queued';
export type Status = Workflow.WorkflowStatus | typeof QUEUED_STATUS;

export interface ExecutionSummary extends Workflow.WorkflowExecutionSummary {
	global_runner_id?: string;
	runner_ids?: string[];
	runner_records?: Array<MobileRunnersResponse>;
	queue?: {
		ticket_id: string;
		position: number;
		line_len: number;
		runner_ids: string[];
	};
	results?: Array<{
		video: string;
		screenshot: string;
	}>;
}

const baseUrl = '/api/pipeline/list-workflows';

export async function listAllGroupedByPipelineId(options = { fetch }) {
	return pb.send<Record<string, ExecutionSummary[]>>(baseUrl, {
		method: 'GET',
		fetch: options.fetch
	});
}

export async function list(pipelineId: string, options = { fetch }) {
	return pb.send<ExecutionSummary[]>(`${baseUrl}/${pipelineId}`, {
		method: 'GET',
		fetch: options.fetch
	});
}

//

export const LIMIT_PARAM = 'limit';
export const OFFSET_PARAM = 'offset';

export async function listAll(options: {
	fetch?: typeof fetch;
	status?: string | null;
	limit?: number;
	offset?: number;
}) {
	const params = new URLSearchParams();
	if (options.status) {
		params.set(Workflow.WORKFLOW_STATUS_QUERY_PARAM, options.status);
	}
	if (options.limit !== undefined) {
		params.set(LIMIT_PARAM, String(options.limit));
	}
	if (options.offset !== undefined) {
		params.set(OFFSET_PARAM, String(options.offset));
	}
	const query = params.toString() ? `?${params.toString()}` : '';
	return pb.send<ExecutionSummary[]>('/api/pipeline/list-results' + query, {
		method: 'GET',
		fetch: options.fetch,
		requestKey: null
	});
}
