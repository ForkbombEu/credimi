// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pipe, String } from 'effect';
import { stringify } from 'yaml';

import { pb } from '@/pocketbase';

import type { EnrichedStep } from './steps-builder/steps-builder.svelte';

import {
	DEEPLINK_STEP_ID_PLACEHOLDER,
	type ActivityOptions,
	type EnrichedPipeline,
	type Pipeline,
	type PipelineStep
} from './types';

/* Fetching pipeline */

export async function fetchPipeline(id: string, options = { fetch }): Promise<EnrichedPipeline> {
	const pipeline = await pb.collection('pipelines').getOne(id, { fetch: options.fetch });
	return {
		metadata: pipeline,
		activity_options: pipeline.runtime.temporal.activity_options as ActivityOptions,
		steps: pipeline.steps as EnrichedStep[]
	};
}

/* YAML processing */

export function createPipelineYaml(
	name: string,
	steps: PipelineStep[],
	activity_options: ActivityOptions
): string {
	const linkedSteps = linkIds(steps);

	const pipeline: Pipeline = {
		name,
		runtime: {
			temporal: {
				activity_options
			}
		},
		steps: linkedSteps
	};

	return pipe(
		stringify(pipeline),
		// Adding spaces
		addNewlineBefore('runtime:'),
		addNewlineBefore('steps:'),
		addNewlineBefore('  - use:'),
		// Correcting first step newline
		replaceWith('\n  - use:', (t) => t.replace('\n', ''), false)
	);
}

function linkIds(steps: PipelineStep[]): PipelineStep[] {
	for (const [index, step] of steps.entries()) {
		if (!(step.use === 'mobile-automation')) continue;

		if (!step.with.parameters) continue;
		if (!('deeplink' in step.with.parameters)) continue;

		const previousStep = steps
			.slice(0, index)
			.toReversed()
			.filter((s) => s.use != 'mobile-automation')
			.at(0);

		if (!previousStep || !('id' in previousStep)) continue;

		let deeplinkPath = '.outputs';
		if (previousStep.use === 'conformance-check') {
			deeplinkPath += '.deeplink';
		}

		step.with.parameters.deeplink = step.with.parameters.deeplink.replace(
			DEEPLINK_STEP_ID_PLACEHOLDER,
			previousStep.id + deeplinkPath
		);
	}
	return steps;
}

// Utils

function addNewlineBefore(token: string, all = true) {
	return replaceWith(token, (token) => `\n${token}`, all);
}

function replaceWith(token: string, transform: (token: string) => string, all = true) {
	if (all) {
		return String.replaceAll(token, transform(token));
	} else {
		return String.replace(token, transform(token));
	}
}
