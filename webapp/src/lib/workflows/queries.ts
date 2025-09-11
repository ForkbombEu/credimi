import { stat } from './../../../../pb_data/types.d';
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { ListMyChecksResponseSchema, type ListMyChecksResponse, type WorkflowExecution } from './../credimiClient.generated';
import type { WorkflowStatusType } from '$lib/temporal';

import { toWorkflowExecution, type HistoryEvent } from '@forkbombeu/temporal-ui';
import { Array, String } from 'effect';
import { z } from 'zod';

import { pb } from '@/pocketbase';
import { warn } from '@/utils/other';

import {
	workflowExecutionInfoSchema,
	workflowResponseSchema,
	type WorkflowResponse
} from './types';
import { client } from '$lib/prova';

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
		const data = await client.listMyChecks({fetch: fetchFn, ...statuses})
		const parsed = data.executions.map((exec) => workflowResponseToExecution({ workflowExecutionInfo: exec }));
		const errors = parsed.filter((execution) => execution instanceof Error);
		if (errors.length > 0) warn(errors);

		const executions = Array.difference(parsed, errors) as WorkflowExecution[];
		return executions;
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
