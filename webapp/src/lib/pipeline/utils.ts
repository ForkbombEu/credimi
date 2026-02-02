// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath, runWithLoading } from '$lib/utils';
import { toast } from 'svelte-sonner';
import { parse, stringify } from 'yaml';

import type { MobileRunnersResponse, PipelinesResponse } from '@/pocketbase/types';

import { goto, m } from '@/i18n';
import { pb } from '@/pocketbase';

import type { Pipeline } from './types';

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
		fn: async () => {
			const parsedYaml = parsePipelineYaml(pipeline.yaml);
			const runnerType = getPipelineRunnerType(pipeline);

			if (runnerType === 'global') {
				const runner = getPipelineRunner(pipeline.id);
				if (!runner) throw new Error('No runner found');
				if (parsedYaml.runtime) parsedYaml.runtime.global_runner_id = runner;
				else parsedYaml.runtime = { global_runner_id: runner };
			}

			return await pb.send('/api/pipeline/start', {
				method: 'POST',
				body: {
					pipeline_identifier: getPath(pipeline),
					yaml: stringify(parsedYaml)
				}
			});
		},
		showSuccessToast: false
	});

	if (result?.result) {
		const { workflowId, workflowRunId } = result.result;
		const workflowUrl =
			workflowId && workflowRunId
				? `/my/tests/runs/${workflowId}/${workflowRunId}`
				: undefined;

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
}
