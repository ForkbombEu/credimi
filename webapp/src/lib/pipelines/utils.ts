// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath, runWithLoading } from '$lib/utils';
import { parse } from 'yaml';
import { toast } from 'svelte-sonner';

import type { PipelinesResponse } from '@/pocketbase/types';

import { goto, m } from '@/i18n';
import { pb } from '@/pocketbase';

//

const PIPELINES_RUNNERS_STORAGE_KEY = 'pipelines_runners_config';

type PipelinesRunnersConfig = Record<string, string>;

/**
 * Parse the pipeline yaml. If there is no key `runner_id` inside mobile automation steps,
 * then it's global. otherwise it's specific
 */
export function getPipelineRunnerType(pipeline: PipelinesResponse): 'global' | 'specific' {
	try {
		const yaml = parse(pipeline.yaml);
		const steps = yaml?.steps ?? [];
		
		// Check all mobile-automation steps
		let hasMobileAutomationSteps = false;
		for (const step of steps) {
			if (step.use === 'mobile-automation') {
				hasMobileAutomationSteps = true;
				// If any mobile-automation step has a runner_id, it's specific
				if (step.with?.config?.runner_id) {
					return 'specific';
				}
			}
		}
		
		// If there are no mobile-automation steps, or none have runner_id, it's global
		return 'global';
	} catch (error) {
		// If parsing fails, assume global
		return 'global';
	}
}

/**
 * Set the pipeline runner in localStorage
 */
export function setPipelineRunner(pipeline: PipelinesResponse, runner: string): void {
	try {
		// Get existing config or create new one
		let config: PipelinesRunnersConfig = {};
		const stored = localStorage.getItem(PIPELINES_RUNNERS_STORAGE_KEY);
		if (stored) {
			config = JSON.parse(stored);
		}
		
		// Set the runner for this pipeline
		config[pipeline.id] = runner;
		
		// Save back to localStorage
		localStorage.setItem(PIPELINES_RUNNERS_STORAGE_KEY, JSON.stringify(config));
	} catch (error) {
		console.error('Failed to set pipeline runner:', error);
	}
}

/**
 * Get the pipeline runner from localStorage
 */
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

/**
 * Run a pipeline with optional global_runner_id
 */
export async function runPipeline(
	pipeline: PipelinesResponse,
	options?: { global_runner_id?: string }
) {
	const result = await runWithLoading({
		fn: async () => {
			// Parse the existing YAML
			const parsedYaml = parse(pipeline.yaml);
			
			// Add global_runner_id to runtime if provided
			if (options?.global_runner_id) {
				if (!parsedYaml.runtime) {
					parsedYaml.runtime = {};
				}
				parsedYaml.runtime.global_runner_id = options.global_runner_id;
			}
			
			// Convert back to YAML string
			const { stringify } = await import('yaml');
			const modifiedYaml = stringify(parsedYaml);
			
			return await pb.send('/api/pipeline/start', {
				method: 'POST',
				body: {
					yaml: modifiedYaml,
					pipeline_identifier: getPath(pipeline)
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
