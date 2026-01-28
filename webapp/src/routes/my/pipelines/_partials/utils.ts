// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath, runWithLoading } from '$lib/utils';
import { toast } from 'svelte-sonner';
import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';

import type { PipelinesResponse } from '@/pocketbase/types';

import { goto, m } from '@/i18n';
import { pb } from '@/pocketbase';

//

interface PipelineYAML {
	global_runner_id?: string;
	steps?: Array<{
		use: string;
		with?: {
			runner_id?: string;
			[key: string]: any;
		};
	}>;
}

/**
 * Check if a pipeline requires a global runner to be selected
 * Returns true if:
 * - Pipeline has mobile-automation steps
 * - No global_runner_id is set
 * - Not all mobile-automation steps have runner_id
 */
export function pipelineRequiresGlobalRunner(pipelineYaml: string): boolean {
	try {
		const parsed = parseYaml(pipelineYaml) as PipelineYAML;
		
		// If global_runner_id is already set, no need for selection
		if (parsed.global_runner_id) {
			return false;
		}

		const mobileSteps = parsed.steps?.filter(step => step.use === 'mobile-automation') || [];
		
		// If no mobile automation steps, no runner needed
		if (mobileSteps.length === 0) {
			return false;
		}

		// Check if all mobile steps have runner_id
		const allHaveRunner = mobileSteps.every(step => step.with?.runner_id);
		
		// If not all have runner, we need global runner selection
		return !allHaveRunner;
	} catch (e) {
		console.error('Failed to parse pipeline YAML:', e);
		return false;
	}
}

/**
 * Get the stored runner for a pipeline from localStorage
 */
export function getStoredRunner(pipelineId: string): string | undefined {
	if (typeof window === 'undefined') return undefined;
	
	try {
		const stored = localStorage.getItem('pipeline_runners');
		if (!stored) return undefined;
		
		const runners = JSON.parse(stored) as Record<string, string>;
		return runners[pipelineId];
	} catch (e) {
		console.error('Failed to get stored runner:', e);
		return undefined;
	}
}

/**
 * Store a runner selection for a pipeline in localStorage
 */
export function storeRunner(pipelineId: string, runnerPath: string): void {
	if (typeof window === 'undefined') return;
	
	try {
		const stored = localStorage.getItem('pipeline_runners');
		const runners = stored ? (JSON.parse(stored) as Record<string, string>) : {};
		runners[pipelineId] = runnerPath;
		localStorage.setItem('pipeline_runners', JSON.stringify(runners));
	} catch (e) {
		console.error('Failed to store runner:', e);
	}
}

export async function runPipeline(pipeline: PipelinesResponse, globalRunnerPath?: string) {
	const result = await runWithLoading({
		fn: async () => {
			// Build the YAML with global_runner_id if provided
			let yamlToSend = pipeline.yaml;
			
			if (globalRunnerPath) {
				try {
					const parsed = parseYaml(pipeline.yaml) as PipelineYAML;
					parsed.global_runner_id = globalRunnerPath;
					yamlToSend = stringifyYaml(parsed);
				} catch (e) {
					console.error('Failed to add global_runner_id to YAML:', e);
				}
			}

			return await pb.send('/api/pipeline/start', {
				method: 'POST',
				body: {
					yaml: yamlToSend,
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
