// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { String } from 'effect';

import { pb } from '@/pocketbase';
import { warn } from '@/utils/other';

import type { FetchWorkflowsResponse, WorkflowExecutionSummary } from './queries.types';
import type { WorkflowExecutionInfo, WorkflowResponse, WorkflowStatus } from './types';

import { toReadableWorkflowStatus } from './status';
import { workflowResponseSchema } from './types';

//

export const WORKFLOW_STATUS_QUERY_PARAM = 'status';

const WORKFLOW_LIST_API = '/api/list-workflows';
const WORKFLOW_DETAILS_API = '/api/my/workflows';

const workflowApi = (workflowId: string, runId: string) =>
	`${WORKFLOW_DETAILS_API}/${workflowId}/runs/${runId}`;

//

type FetchWorkflowsOptions = {
	fetch?: typeof fetch;
	status?: string | null;
};

/** Describe payload fields Credimi uses on the run page (no Temporal UI model). */
export type WorkflowRunDetails = {
	info: WorkflowExecutionInfo;
	status: WorkflowStatus;
	failure_reason?: string;
	devices: NonNullable<WorkflowResponse['devices']>;
};

export async function fetchWorkflows(
	options: FetchWorkflowsOptions = {}
): Promise<WorkflowExecutionSummary[] | Error> {
	const { fetch: fetchFn = fetch, status } = options;

	let url = WORKFLOW_LIST_API;
	if (status) {
		const formattedStatus = String.pascalToSnake(status);
		url += `?${WORKFLOW_STATUS_QUERY_PARAM}=${formattedStatus}`;
	}

	return tryPromise(async () => {
		const data: FetchWorkflowsResponse = await pb.send(url, {
			method: 'GET',
			fetch: fetchFn,
			requestKey: null
		});

		return data.executions ?? [];
	}, 'Failed to fetch user workflows');
}

export async function fetchWorkflowExecution(
	workflowId: string,
	runId: string,
	options = { fetch }
): Promise<WorkflowRunDetails | Error> {
	return tryPromise(async () => {
		const data = await pb.send(workflowApi(workflowId, runId), {
			method: 'GET',
			fetch: options.fetch
		});
		const parsed = workflowResponseSchema.parse(data);
		return {
			info: parsed.workflowExecutionInfo,
			status: toReadableWorkflowStatus(parsed.workflowExecutionInfo.status),
			failure_reason: parsed.failure_reason,
			devices: parsed.devices ?? []
		};
	}, 'Failed to fetch workflow');
}

async function tryPromise<T>(fn: () => Promise<T>, errorMessage?: string): Promise<T | Error> {
	try {
		return await fn();
	} catch (error) {
		warn(errorMessage, error);
		if (error instanceof Error) return error;
		else return new Error(errorMessage);
	}
}
