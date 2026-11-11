// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowStatus } from '$lib/workflows/types';

import { resolve } from '$app/paths';

import { pb } from '@/pocketbase';

//

export type ExecutionStepKind = 'activity' | 'block' | 'workflow';

export type ExecutionRef = { workflowId: string; runId: string };

/** One started main step of a pipeline run, read from its Temporal history. */
export type ExecutionStep = {
	step_id: string;
	kind: ExecutionStepKind;
	type: string;
	display_name: string;
	status: WorkflowStatus;
	start_time?: string;
	end_time?: string;
	duration?: string;
	attempt?: number;
	max_attempts?: number;
	failure_reason?: string;
	/** The event that started the step, in the history of the run that ran it. */
	history_event: { workflow_id: string; run_id: string; event_id: number };
	/** The child workflow run of workflow and block steps. */
	execution?: ExecutionRef;
	has_logs?: boolean;
	has_qr?: boolean;
};

export async function listExecutionSteps(
	pipelineId: string,
	run: ExecutionRef,
	block?: ExecutionRef
): Promise<ExecutionStep[]> {
	const query = block
		? `?${new URLSearchParams({ block_workflow_id: block.workflowId, block_run_id: block.runId })}`
		: '';
	const response = await pb.send<{ steps: ExecutionStep[] }>(
		`/api/pipeline/executions/${pipelineId}/${encodeURIComponent(run.workflowId)}/${encodeURIComponent(run.runId)}/steps${query}`,
		{ method: 'GET', requestKey: null }
	);
	return response.steps;
}

export const STEPS_INITIAL_LIMIT = 100;

const ALWAYS_SHOWN_STATUSES: WorkflowStatus[] = ['Failed', 'Running', 'TimedOut'];

/**
 * Long step lists show the first `STEPS_INITIAL_LIMIT` steps, plus every later step that
 * failed, timed out or is still running, in their original order.
 */
export function visibleSteps(steps: ExecutionStep[], showAll: boolean): ExecutionStep[] {
	if (showAll || steps.length <= STEPS_INITIAL_LIMIT) return steps;
	return steps.filter(
		(step, index) => index < STEPS_INITIAL_LIMIT || ALWAYS_SHOWN_STATUSES.includes(step.status)
	);
}

/**
 * Workflow and block steps open their own run page; activity steps open the run that ran
 * them at the event that started them.
 */
export function stepDetailsHref(step: ExecutionStep): string {
	if (step.kind !== 'activity' && step.execution) {
		return resolve('/my/tests/runs/[workflow_id]/[run_id]', {
			workflow_id: step.execution.workflowId,
			run_id: step.execution.runId
		});
	}
	const runPage = resolve('/my/tests/runs/[workflow_id]/[run_id]', {
		workflow_id: step.history_event.workflow_id,
		run_id: step.history_event.run_id
	});
	return `${runPage}?event=${step.history_event.event_id}`;
}
