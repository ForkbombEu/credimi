// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { type Pipeline, type PipelineFinally, type PipelineStep } from '$lib/pipeline/types';
import * as _ from 'lodash';
import { stringify } from 'yaml';

import type { EnrichedFollowUp } from './get-enriched-pipeline.js';
import type { RuntimeOptions } from './runtime-options-form/runtime-options-form.svelte.js';

import { getFinallyConditions } from './pipeline-finally.js';
import { assignStepId, seedIdCounters } from './pipeline-step-ids.js';
import { getConfigByTypeOrThrow } from './steps';
import { formatPipelineYamlDocument } from './yaml-format.js';
import { orderMapKeysByValueKind, orderStepKeys } from './yaml-key-order.js';

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

	return formatPipelineYamlDocument(stringify(pipeline));
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
