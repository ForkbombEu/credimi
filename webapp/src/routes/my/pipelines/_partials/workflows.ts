// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';
import { getRecordByCanonifiedPath } from '$lib/canonify';

import { pb } from '@/pocketbase';
import type { MobileRunnersResponse } from '@/pocketbase/types';

//

const baseUrl = '/api/pipeline/list-workflows';

export type PipelinesWorkflows = Record<string, WorkflowExecutionSummary[]>;

const runnerNameCache = new Map<string, string>();

function normalizeRunnerId(runnerId: string) {
	return runnerId.trim();
}

function collectRunnerIds(workflow: WorkflowExecutionSummary): string[] {
	const ids = new Set<string>();

	for (const runnerId of workflow.runner_ids ?? []) {
		const normalized = normalizeRunnerId(runnerId);
		if (normalized) ids.add(normalized);
	}

	const globalRunnerId = normalizeRunnerId(workflow.global_runner_id ?? '');
	if (globalRunnerId) ids.add(globalRunnerId);

	return Array.from(ids);
}

async function resolveRunnerName(
	runnerId: string,
	fetchFn: typeof fetch
): Promise<string> {
	const cached = runnerNameCache.get(runnerId);
	if (cached !== undefined) {
		return cached;
	}

	const record = await getRecordByCanonifiedPath<MobileRunnersResponse>(runnerId, {
		fetch: fetchFn
	});
	const name = record instanceof Error ? '' : (record.name ?? '');
	runnerNameCache.set(runnerId, name);
	return name;
}

async function attachRunnerNames(
	workflow: WorkflowExecutionSummary,
	fetchFn: typeof fetch
): Promise<void> {
	const runnerIds = collectRunnerIds(workflow);
	if (runnerIds.length > 0) {
		const names = await Promise.all(
			runnerIds.map((runnerId) => resolveRunnerName(runnerId, fetchFn))
		);
		workflow.runner_names = Array.from(new Set(names.filter((name) => name !== '')));
	} else {
		workflow.runner_names = [];
	}

	for (const child of workflow.children ?? []) {
		await attachRunnerNames(child, fetchFn);
	}
}

async function resolveRunnerNames(
	workflows: WorkflowExecutionSummary[],
	fetchFn: typeof fetch
): Promise<void> {
	for (const workflow of workflows) {
		await attachRunnerNames(workflow, fetchFn);
	}
}

export async function getAllPipelinesWorkflows(options = { fetch }): Promise<PipelinesWorkflows> {
	const response = await pb.send(baseUrl, {
		method: 'GET',
		fetch: options.fetch
	});

	const workflows = response as PipelinesWorkflows;
	for (const executions of Object.values(workflows)) {
		await resolveRunnerNames(executions, options.fetch);
	}
	return workflows;
}

export async function getPipelineWorkflows(
	pipelineId: string,
	options = { fetch }
): Promise<WorkflowExecutionSummary[]> {
	const response = await pb.send(`${baseUrl}/${pipelineId}`, {
		method: 'GET',
		fetch: options.fetch
	});
	const workflows = response as WorkflowExecutionSummary[];
	await resolveRunnerNames(workflows, options.fetch);
	return workflows;
}
