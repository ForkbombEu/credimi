// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowExecution } from '@forkbombeu/temporal-ui/dist/types/workflows';

import { toWorkflowExecution, type HistoryEvent } from '@forkbombeu/temporal-ui';
import { String } from 'effect';
import { z } from 'zod/v3';

import { pb } from '@/pocketbase';
import { warn } from '@/utils/other';

import type { FetchWorkflowsResponse, WorkflowExecutionSummary } from './queries.types';

import { workflowResponseSchema, type WorkflowResponse } from './types';

//

export const WORKFLOW_STATUS_QUERY_PARAM = 'status';

const WORKFLOWS_API = '/api/compliance/checks';

const workflowApi = (workflowId: string, runId: string) =>
	`${WORKFLOWS_API}/${workflowId}/${runId}`;

//

type FetchWorkflowsOptions = {
	fetch?: typeof fetch;
	status?: string | null;
};

export async function fetchWorkflows(
	options: FetchWorkflowsOptions = {}
): Promise<WorkflowExecutionSummary[] | Error> {
	// const test = await import('./queries.test.json');
	// return test.default.executions;

	const { fetch: fetchFn = fetch, status } = options;

	let url = WORKFLOWS_API;
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
): Promise<WorkflowExecution | Error> {
	return tryPromise(async () => {
		const data = await pb.send(workflowApi(workflowId, runId), {
			method: 'GET',
			fetch: options.fetch
		});
		const parsed = workflowResponseSchema.parse(data);
		return workflowResponseToExecution(parsed);
	}, 'Failed to fetch workflow');
}

export async function fetchWorkflowHistory(
	workflowId: string,
	runId: string,
	options = { fetch }
): Promise<HistoryEvent[] | Error> {
	return tryPromise(async () => {
		const data = await pb.send(`${workflowApi(workflowId, runId)}/history`, {
			method: 'GET',
			fetch: options.fetch
		});
		const schema = z.array(z.record(z.unknown()));
		return schema.parse(data) as HistoryEvent[];
	}, 'Failed to fetch workflow history');
}

// Private

function workflowResponseToExecution(data: WorkflowResponse): WorkflowExecution | Error {
	return tryFn(() => {
		// @ts-expect-error Slight type mismatch
		const workflowExecution = toWorkflowExecution(data);

		/* HACK */
		// canBeTerminated a property of workflow object is a getter that requires a svelte `store` to work
		// by removing it, we can avoid the store dependency and solve a svelte error about state not updating
		Object.defineProperty(workflowExecution, 'canBeTerminated', {
			value: true
		});

		return workflowExecution;
	}, 'Failed to convert workflow response to execution');
}

function tryFn<T>(fn: () => T, errorMessage?: string): T | Error {
	try {
		return fn();
	} catch (error) {
		warn(errorMessage, error);
		if (error instanceof Error) return error;
		else return new Error(errorMessage);
	}
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
