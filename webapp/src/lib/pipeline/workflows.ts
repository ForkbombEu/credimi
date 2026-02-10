// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';

import type { MobileRunnersResponse } from '@/pocketbase/types';

import { pb } from '@/pocketbase';

import SmallTable from './workflows-table-small.svelte';
import Table from './workflows-table.svelte';
export { SmallTable, Table };

//

export interface ExecutionSummary extends WorkflowExecutionSummary {
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

export async function listAll(options = { fetch }) {
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
