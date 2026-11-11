// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { type Pipeline, type PipelineFinally, type PipelineStep } from '$lib/pipeline/types';
import { serializeYaml } from '$lib/pipeline/utils';
import * as _ from 'lodash';

import type { EnrichedFollowUp } from '../get-enriched-pipeline.js';
import type { RuntimeOptions } from '../runtime-options-form/runtime-options-form.svelte.js';

import { getConfigByTypeOrThrow } from '../steps';
import { getFinallyConditions } from './finally.js';
import { formatPipelineYamlDocument } from './format.js';
import { orderMapKeysByValueKind, orderStepKeys } from './key-order.js';
import { assignStepId, seedIdCounters } from './step-ids.js';

export function createPipelineYaml(
	name: string,
	steps: PipelineStep[],
	runtime: RuntimeOptions,
	followUps: EnrichedFollowUp[] = []
): string {
	// Cloning because editing ids and linking steps modify the original steps array
	const clonedSteps = _.cloneDeep(steps);
	const clonedFollowUps = _.cloneDeep(followUps);
	const idCounters = seedIdCounters([
		...clonedSteps,
		...clonedFollowUps.map(({ step }) => step[0] as PipelineStep)
	]);

	const processedSteps = clonedSteps.map((step, index) => {
		if (step.use === 'debug') {
			return orderStepKeys(step);
		}
		const config = getConfigByTypeOrThrow(step.use);
		if ('id' in step) {
			assignStepId(step, idCounters);
		}
		// Builder state only marks "linked to the previous step": rewrite it to the previous
		// step's (already assigned) id, or drop it when that step cannot be linked to.
		if ('linked_to' in step && step.linked_to) {
			const previous = clonedSteps[index - 1];
			if (previous && previous.use !== 'debug' && previous.id) {
				step.linked_to = previous.id;
			} else {
				delete step.linked_to;
			}
		}
		if (config.linkProcedure && 'with' in step) {
			config.linkProcedure?.(step.with, clonedSteps.slice(0, index));
		}
		return orderStepKeys(step);
	});

	const pipeline: Pipeline = {
		name,
		runtime: orderMapKeysByValueKind(runtime) as RuntimeOptions,
		steps: processedSteps
	};
	const finallyDefinition = createFinallyDefinition(clonedFollowUps, idCounters);
	if (finallyDefinition) {
		pipeline.finally = finallyDefinition;
	}

	return formatPipelineYamlDocument(serializeYaml(pipeline));
}

function createFinallyDefinition(
	followUps: EnrichedFollowUp[],
	idCounters: Map<string, number>
): Exclude<PipelineFinally, unknown[]> | undefined {
	const definition: Exclude<PipelineFinally, unknown[]> = {};

	for (const condition of getFinallyConditions()) {
		const steps = followUps
			.filter((followUp) => followUp.condition === condition)
			.map(({ step }) => {
				const rawStep = step[0] as PipelineStep;
				assignStepId(rawStep, idCounters);
				return toFinallyStep(rawStep);
			});
		if (steps.length > 0) {
			definition[condition] = steps as never;
		}
	}

	return Object.keys(definition).length > 0 ? definition : undefined;
}

function toFinallyStep(step: PipelineStep) {
	const finallyStep = { ...step } as PipelineStep & {
		continue_on_error?: boolean;
	};
	delete finallyStep.continue_on_error;
	return orderStepKeys(finallyStep);
}
