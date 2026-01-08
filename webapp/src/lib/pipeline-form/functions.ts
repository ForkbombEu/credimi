// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pipe, String } from 'effect';
import { parse, stringify } from 'yaml';

import type { PipelinesResponse } from '@/pocketbase/types';

import type { EnrichedStep } from './steps-builder/steps-builder.svelte.js';

import { configs } from './steps';
import {
	DEEPLINK_STEP_ID_PLACEHOLDER,
	type ActivityOptions,
	type EnrichedPipeline,
	type Pipeline,
	type PipelineStep,
	type PipelineStepType
} from './types';

/* Fetching pipeline */

export async function enrichPipeline(pipelineRecord: PipelinesResponse): Promise<EnrichedPipeline> {
	const pipelineYaml = parse(pipelineRecord.yaml) as Pipeline;
	const steps = pipelineYaml.steps ?? [];

	const enrichedSteps: EnrichedStep[] = [];
	for (const step of steps) {
		if (step.use === 'debug') {
			enrichedSteps.push([step, {}]);
		} else {
			try {
				const config = configs.find((c) => c.id === step.use);
				if (!config) throw new Error(`Unknown step type: ${step.use}`);
				const data = await config.deserialize(step.with);
				enrichedSteps.push([step, data]);
			} catch (e) {
				console.error(step);
				console.error(e);
				// TODO: Push 404 step
				enrichedSteps.push([step, {}]);
			}
		}
	}

	return {
		metadata: pipelineRecord,
		activity_options: pipelineYaml.runtime?.temporal?.activity_options,
		steps: enrichedSteps
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
		addNewlineBefore('  - id:'),
		// Correcting first step newline
		replaceWith('steps:\n\n', (t) => t.replace('\n\n', '\n'), false)
	);
}

const unlinkableSteps: PipelineStepType[] = ['mobile-automation', 'debug'];

function linkIds(steps: PipelineStep[]): PipelineStep[] {
	for (const [index, step] of steps.entries()) {
		if (!(step.use === 'mobile-automation')) continue;
		if (!step.with.parameters) continue;
		if (!('deeplink' in step.with.parameters)) continue;

		const previousStep = steps
			.slice(0, index)
			.toReversed()
			.filter((s) => !unlinkableSteps.includes(s.use))
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
