// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath, runWithLoading } from '$lib/utils';
import { toast } from 'svelte-sonner';
import { parse, stringify } from 'yaml';

import type { MobileRunnersResponse, PipelinesResponse } from '@/pocketbase/types';
import { getExceptionMessage } from '@/utils/errors';

import { goto, m } from '@/i18n';
import { pb } from '@/pocketbase';

import type { Pipeline } from './types';

type PipelineRunnerType = 'global' | 'specific' | 'none';

type PipelineQueueStatusResponse = {
	ticket_id: string;
	runner_ids?: string[];
	status: 'queued' | 'starting' | 'running' | 'failed' | 'canceled' | 'not_found';
	position: number;
	line_len: number;
	workflow_id?: string;
	run_id?: string;
	workflow_namespace?: string;
	error_message?: string;
};

//

export function parsePipelineYaml(yaml: string): Pipeline {
	return parse(yaml) as Pipeline;
}

function getRunnerTypeForYaml(pipeline: Pipeline): PipelineRunnerType {
	const steps = (pipeline?.steps ?? []).filter((step) => step.use === 'mobile-automation');

	if (steps.length === 0) return 'none';

	const areAllStepsSpecific = steps.every((step) => step.with.runner_id);
	if (areAllStepsSpecific) return 'specific';

	const areSomeStepsSpecific = steps.some((step) => step.with.runner_id);
	if (areSomeStepsSpecific) throw new Error('Mixed runner types');

	return 'global';
}

export function getPipelineRunnerType(pipeline: PipelinesResponse): PipelineRunnerType {
	return getRunnerTypeForYaml(parsePipelineYaml(pipeline.yaml));
}

/* Runners configuration storage */

const PIPELINES_RUNNERS_STORAGE_KEY = 'pipelines_runners_config';

type PipelinesRunnersConfig = Record<string, string>;

export function setPipelineRunner(
	pipeline: PipelinesResponse,
	runner: MobileRunnersResponse
): void {
	try {
		let config: PipelinesRunnersConfig = {};
		const stored = localStorage.getItem(PIPELINES_RUNNERS_STORAGE_KEY);
		if (stored) config = JSON.parse(stored);

		config[pipeline.id] = getPath(runner);
		localStorage.setItem(PIPELINES_RUNNERS_STORAGE_KEY, JSON.stringify(config));
	} catch (error) {
		console.error('Failed to set pipeline runner:', error);
	}
}

export function getPipelineRunner(pipelineId: string): string | undefined {
	try {
		const stored = localStorage.getItem(PIPELINES_RUNNERS_STORAGE_KEY);
		if (!stored) return undefined;

		const config: PipelinesRunnersConfig = JSON.parse(stored);
		return config[pipelineId];
	} catch (error) {
		console.error('Failed to get pipeline runner:', error);
		return undefined;
	}
}

/* Running */

function showWorkflowStartedToast(workflowId?: string, workflowRunId?: string) {
	const workflowUrl =
		workflowId && workflowRunId ? `/my/tests/runs/${workflowId}/${workflowRunId}` : undefined;

	toast.success(m.Pipeline_started_successfully(), {
		description: m.View_workflow_details(),
		duration: 10000,
		...(workflowUrl && {
			action: {
				label: m.View(),
				onClick: () => goto(workflowUrl)
			}
		})
	});
}

function queueStatusUrl(ticketId: string, runnerIds: string[]) {
	const params = new URLSearchParams();
	params.set('runner_ids', runnerIds.join(','));
	return `/api/pipeline/queue/${ticketId}?${params.toString()}`;
}

function formatQueueToastMessage(position: number, lineLen: number, runnerCount: number) {
	const displayPosition = position + 1;
	const displayLineLen = Math.max(lineLen, displayPosition);
	const suffix = runnerCount > 1 ? ' (multiple runners)' : '';
	return `Queued: position ${displayPosition} of ${displayLineLen}${suffix}`;
}

export async function runPipeline(pipeline: PipelinesResponse) {
	const parsedYaml = parsePipelineYaml(pipeline.yaml);
	const runnerType = getRunnerTypeForYaml(parsedYaml);
	const pipelineIdentifier = getPath(pipeline);

	if (runnerType === 'global') {
		const runner = getPipelineRunner(pipeline.id);
		if (!runner) throw new Error('No runner found');
		if (parsedYaml.runtime) parsedYaml.runtime.global_runner_id = runner;
		else parsedYaml.runtime = { global_runner_id: runner };
	}

	if (runnerType === 'none') {
		const result = await runWithLoading({
			fn: async () =>
				pb.send('/api/pipeline/start', {
					method: 'POST',
					body: {
						pipeline_identifier: pipelineIdentifier,
						yaml: stringify(parsedYaml)
					}
				}),
			showSuccessToast: false
		});

		if (result?.result) {
			showWorkflowStartedToast(result.result.workflowId, result.result.workflowRunId);
		}
		return;
	}

	const enqueueResult = await runWithLoading({
		fn: async () => {
			try {
				const value = await pb.send<PipelineQueueStatusResponse>('/api/pipeline/queue', {
					method: 'POST',
					body: {
						pipeline_identifier: pipelineIdentifier,
						yaml: stringify(parsedYaml)
					}
				});
				return { ok: true as const, value };
			} catch (error) {
				return { ok: false as const, error: getExceptionMessage(error) };
			}
		},
		showSuccessToast: false
	});

	if (!enqueueResult) return;
	if (!enqueueResult.ok) {
		toast.error(enqueueResult.error);
		return;
	}

	const runnerIds = enqueueResult.value.runner_ids ?? [];
	if (!enqueueResult.value.ticket_id || runnerIds.length === 0) {
		toast.error('Failed to start queue');
		return;
	}

	if (enqueueResult.value.status === 'running') {
		showWorkflowStartedToast(enqueueResult.value.workflow_id, enqueueResult.value.run_id);
		return;
	}

	if (enqueueResult.value.status === 'failed') {
		toast.error(enqueueResult.value.error_message ?? 'Pipeline failed to start');
		return;
	}

	const ticketId = enqueueResult.value.ticket_id;
	let polling = true;
	let cancelInFlight = false;
	let queueToastId: string | number | undefined;

	const stopPolling = () => {
		polling = false;
	};

	const dismissQueueToast = () => {
		if (queueToastId) toast.dismiss(queueToastId);
	};

	const finishQueue = (action: () => void) => {
		stopPolling();
		dismissQueueToast();
		action();
	};

	const cancelQueuedRun = async () => {
		if (cancelInFlight) return;
		cancelInFlight = true;
		try {
			const cancelStatus = await pb.send<PipelineQueueStatusResponse>(
				queueStatusUrl(ticketId, runnerIds),
				{ method: 'DELETE' }
			);
			if (cancelStatus.status === 'running' || cancelStatus.status === 'failed') {
				cancelInFlight = false;
				handleQueueStatus(cancelStatus);
				return;
			}
			if (cancelStatus.status === 'not_found' || cancelStatus.status === 'canceled') {
				finishQueue(() => toast.message('Queue canceled'));
				return;
			}
			cancelInFlight = false;
		} catch (error) {
			cancelInFlight = false;
			toast.error('Failed to cancel queue');
		}
	};

	queueToastId = toast.info(
		formatQueueToastMessage(
			enqueueResult.value.position,
			enqueueResult.value.line_len,
			runnerIds.length
		),
		{
			duration: Infinity,
			action: {
				label: m.Cancel(),
				onClick: cancelQueuedRun
			}
		}
	);

	const handleQueueStatus = (status: PipelineQueueStatusResponse) => {
		if (status.status === 'running') {
			finishQueue(() => showWorkflowStartedToast(status.workflow_id, status.run_id));
			return;
		}
		if (status.status === 'failed') {
			finishQueue(() => toast.error(status.error_message ?? 'Pipeline failed to start'));
			return;
		}
		if (status.status === 'canceled') {
			finishQueue(() => toast.message('Queue canceled'));
			return;
		}
		if (status.status === 'not_found') {
			finishQueue(() => toast.error('Queue ticket not found'));
		}
	};

	const pollQueueStatus = async () => {
		if (!polling) return;
		try {
			const status = await pb.send<PipelineQueueStatusResponse>(
				queueStatusUrl(ticketId, runnerIds),
				{ method: 'GET' }
			);
			handleQueueStatus(status);
		} catch (error) {
			stopPolling();
			dismissQueueToast();
			toast.error('Failed to poll queue');
			return;
		}
		if (polling) {
			setTimeout(pollQueueStatus, 1000);
		}
	};

	void pollQueueStatus();
}
