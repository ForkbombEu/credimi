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
	pipeline_identifier?: string;
	pipeline_name?: string;
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
		log: string;
	}>;
}

const groupedExecutionsUrl = '/api/pipeline/list-executions';

export async function listAllGroupedByPipelineId(options = { fetch }) {
	return pb.send<Record<string, ExecutionSummary[]>>(groupedExecutionsUrl, {
		method: 'GET',
		fetch: options.fetch,
		requestKey: null
	});
}

export async function list(
	pipelineId: string,
	options: {
		fetch?: typeof fetch;
		status?: string | null;
		limit?: number;
		offset?: number;
	} = {}
) {
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

	return pb.send<ExecutionSummary[]>(`${groupedExecutionsUrl}/${pipelineId}${query}`, {
		method: 'GET',
		fetch: options.fetch,
		requestKey: null
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
	const grouped = await listAllGroupedByPipelineId({ fetch: options.fetch });
	const flattened = Object.values(grouped)
		.flat()
		.filter((execution) => !options.status || execution.status === options.status)
		.sort((left, right) => parseExecutionTime(right) - parseExecutionTime(left));

	const offset = options.offset ?? 0;
	const limit = options.limit ?? flattened.length;
	return flattened.slice(offset, offset + limit);
}

function parseExecutionTime(execution: ExecutionSummary): number {
	const value = execution.enqueuedAt ?? execution.startTime;
	if (!value) return 0;

	const localizedMatch = value.match(
		/^(\d{2})\/(\d{2})\/(\d{4}), (\d{2}):(\d{2}):(\d{2})$/
	);
	if (localizedMatch) {
		const [, day, month, year, hour, minute, second] = localizedMatch;
		return new Date(
			Number(year),
			Number(month) - 1,
			Number(day),
			Number(hour),
			Number(minute),
			Number(second)
		).getTime();
	}

	return new Date(value).getTime();
}
