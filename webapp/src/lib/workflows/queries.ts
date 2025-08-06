// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { ListMyChecksResponseSchema, type ListMyChecksResponse } from './../credimiClient';
import { workflowExecutionInfoSchema } from './types';
import type { WorkflowStatusType } from '$lib/temporal';
import { Array, String } from 'effect';
import { z } from 'zod';
import { warn } from '@/utils/other';
import type { WorkflowExecution } from '@forkbombeu/temporal-ui/dist/types/workflows';
import { toWorkflowExecution, type HistoryEvent } from '@forkbombeu/temporal-ui';

//

const WORKFLOWS_API = '/api/compliance/checks';

const workflowApi = (workflowId: string, runId: string) =>
	`${WORKFLOWS_API}/${workflowId}/${runId}`;

//

type FetchWorkflowsOptions = {
	fetch?: typeof fetch;
	statuses?: WorkflowStatusType[];
};

export async function fetchWorkflows(
	options: FetchWorkflowsOptions = {}
): Promise<WorkflowExecution[] | Error> {
	const { fetch: fetchFn = fetch, statuses = [] } = options;

	let url = WORKFLOWS_API;
	if (statuses.length > 0) {
		const formattedStatuses = statuses.map((status) => String.pascalToSnake(status));
		url += `?status=${formattedStatuses.join(',')}`;
	}

	return tryPromise(async () => {
		const data = await pb.send(url, {
			method: 'GET',
			fetch: fetchFn
		});
		const schema = ListMyChecksResponseSchema
		console.log('Fetched workflows data:', data);

		const parsed = schema.parse(data).executions.map(workflowInfoToExecution);
		const errors = parsed.filter((execution) => execution instanceof Error);
		const executions = Array.difference(parsed, errors) as WorkflowExecution[];

		warn(errors);
		return executions;
	}, 'Failed to fetch user workflows');
}

//

export async function fetchWorkflow(
	workflowId: string,
	runId: string,
	options = { fetch }
): Promise<WorkflowExecution | Error> {
	return tryPromise(async () => {
		const data = await pb.send(workflowApi(workflowId, runId), {
			method: 'GET',
			fetch: options.fetch
		});
		// Note: this schema loses some information (see type `WorkflowExecutionAPIResponse`)
		const schema = z.object({
			workflowExecutionInfo: workflowExecutionInfoSchema
		});
		const parsed = schema.parse(data).workflowExecutionInfo;
		return workflowInfoToExecution(parsed);
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

//

function workflowInfoToExecution(data: ListMyChecksResponse["executions"][number]): WorkflowExecution | Error {
	return tryFn(() => {
		const w = toWorkflowExecution({ workflowExecutionInfo: data });

		/* HACK */
		// canBeTerminated a property of workflow object is a getter that requires a svelte `store` to work
		// by removing it, we can avoid the store dependency and solve a svelte error about state not updating
		Object.defineProperty(w, 'canBeTerminated', {
			value: true
		});

		return w;
	}, 'Failed to convert workflow response to execution');
}

//

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
