// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { QueryClient } from '@tanstack/svelte-query';

import type { ExecutionSummary } from './workflows';

export const PIPELINE_LIST_WORKFLOWS_LIMIT = 5;

/** Idle list cards: match prior page-level poll. */
export const PIPELINE_LIST_WORKFLOWS_IDLE_POLL_MS = 30_000;

/** Queued/running rows, or a short window after Start Now / cancel. */
export const PIPELINE_LIST_WORKFLOWS_FAST_POLL_MS = 3_000;

/** Keep fast polling after a mutation so Temporal visibility lag does not wait on the idle interval. */
export const PIPELINE_LIST_WORKFLOWS_MUTATION_BOOST_MS = 8_000;

export const PIPELINE_LIST_WORKFLOWS_QUERY_KEY = 'pipeline-list-workflows' as const;

export function pipelineListWorkflowsQueryKey(pipelineId: string) {
	return [
		PIPELINE_LIST_WORKFLOWS_QUERY_KEY,
		pipelineId,
		{ limit: PIPELINE_LIST_WORKFLOWS_LIMIT }
	] as const;
}

export function invalidatePipelineListWorkflows(queryClient: QueryClient, pipelineId: string) {
	return queryClient.invalidateQueries({
		queryKey: [PIPELINE_LIST_WORKFLOWS_QUERY_KEY, pipelineId]
	});
}

/** True when list rows need a short refetch interval (queued or running). */
export function workflowsNeedFastPoll(workflows: ExecutionSummary[] | undefined | null): boolean {
	if (!workflows?.length) return false;
	return workflows.some((workflow) => Boolean(workflow.queue) || workflow.status === 'Running');
}
