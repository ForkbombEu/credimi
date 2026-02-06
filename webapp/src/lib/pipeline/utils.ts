// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath, runWithLoading } from '$lib/utils';
import { toast } from 'svelte-sonner';
import { parse } from 'yaml';

import type { MobileRunnersResponse, PipelinesResponse } from '@/pocketbase/types';

import { m } from '@/i18n';

import type { Pipeline } from './types';

import { cancelQueueTicket, enqueuePipeline } from './queue';

//

export function parsePipelineYaml(yaml: string): Pipeline {
	return parse(yaml) as Pipeline;
}

export function getPipelineRunnerType(pipeline: PipelinesResponse): 'global' | 'specific' {
	const yaml = parsePipelineYaml(pipeline.yaml);
	const steps = (yaml?.steps ?? []).filter((step) => step.use === 'mobile-automation');

	if (steps.length === 0) return 'global';

	const areAllStepsSpecific = steps.every((step) => step.with.runner_id);
	if (areAllStepsSpecific) return 'specific';

	const areSomeStepsSpecific = steps.some((step) => step.with.runner_id);
	if (areSomeStepsSpecific) throw new Error('Mixed runner types');

	return 'global';
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

export async function runPipeline(pipeline: PipelinesResponse) {
	const result = await runWithLoading({
		fn: () => enqueuePipeline(pipeline),
		showSuccessToast: false
	});

	if (!result) {
		toast.error('Unexpected error');
		return;
	}

	if (result.isErr) {
		toast.error(result.error);
		return;
	}

	const runnerIds = result.value.runner_ids ?? [];
	if (!result.value.ticket_id || runnerIds.length === 0) {
		toast.error(m.Failed_to_enqueue_pipeline());
		return;
	}

	if (result.value.status === 'failed') {
		toast.error(result.value.error_message ?? m.Failed_to_enqueue_pipeline());
		return;
	}

	if (result.value.status === 'queued') {
		toast.info(
			m.Pipeline_queued({ position: result.value.position, line_len: result.value.line_len }),
			{
				duration: 5000,
				action: {
					label: m.Cancel(),
					onClick: async () => {
						const response = await cancelQueueTicket(result.value.ticket_id, runnerIds);
						if (response.isOk) toast.success(m.Pipeline_execution_canceled());
						else toast.error(response.error);
					}
				}
			}
		);
	}
}
