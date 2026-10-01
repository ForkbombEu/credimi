// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	type Pipeline,
	type PipelineFinally,
	type PipelineFinallyCondition,
	type PipelineStep
} from '$lib/pipeline/types';
import { Enrich404Error, type EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
import { Effect, pipe, String } from 'effect';
import * as _ from 'lodash';
import { ClientResponseError } from 'pocketbase';
import slugify from 'slugify';
import { parse, stringify } from 'yaml';

import type { PipelinesResponse } from '@/pocketbase/types';
import type { GenericRecord } from '@/utils/types.js';

import { pb } from '@/pocketbase';
import { getExceptionMessage } from '@/utils/errors.js';

import type { RuntimeOptions } from './runtime-options-form/runtime-options-form.svelte.js';

import { getConfigByTypeOrThrow } from './steps';

/* Fetching pipeline */

/** Cap parallel step enrichment to avoid flooding PocketBase rate limits. */
export const PIPELINE_ENRICH_CONCURRENCY = 12;

export interface EnrichedPipeline {
	record: PipelinesResponse;
	runtime?: RuntimeOptions;
	steps: EnrichedStep[];
	followUps?: EnrichedFollowUp[];
}

export type EnrichedFollowUp = {
	step: EnrichedStep;
	condition: PipelineFinallyCondition;
};

export async function getEnrichedPipeline(
	id: string,
	options = { fetch }
): Promise<EnrichedPipeline> {
	const record = await pb.collection('pipelines').getOne(id, {
		fetch: options.fetch,
		requestKey: null
	});

	const yaml = parse(record.yaml) as Pipeline;
	const steps = yaml.steps ?? [];

	const enrichedSteps = await Effect.runPromise(
		Effect.forEach(steps, (step) => Effect.promise(() => enrichStep(step)), {
			concurrency: PIPELINE_ENRICH_CONCURRENCY
		})
	);
	const followUps = await Effect.runPromise(enrichFollowUps(yaml.finally));

	return {
		record,
		runtime: yaml.runtime,
		steps: enrichedSteps,
		followUps
	};
}

/* YAML processing */

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
			return step;
		}
		const config = getConfigByTypeOrThrow(step.use);
		if ('id' in step) {
			assignStepId(step, idCounters);
		}
		if (config.linkProcedure && 'with' in step) {
			config.linkProcedure?.(step.with, clonedSteps.slice(0, index));
		}
		return step;
	});

	const pipeline: Pipeline = {
		name,
		runtime: runtime,
		steps: processedSteps
	};
	const finallyDefinition = createFinallyDefinition(clonedFollowUps, idCounters);
	if (finallyDefinition) {
		pipeline.finally = finallyDefinition;
	}

	return pipe(
		stringify(pipeline),
		// Adding spaces between top-level sections
		addNewlineBefore('runtime:'),
		addNewlineBefore('steps:'),
		addNewlineBefore('finally:'),
		// Blank line between finally condition buckets (always / on_success / on_failure)
		addNewlineBeforeFinallyCondition('always'),
		addNewlineBeforeFinallyCondition('on_success'),
		addNewlineBeforeFinallyCondition('on_failure'),
		// Blank line before any top-level step list item (exactly two spaces).
		// Matches `- id:` / `- use:` / `- continue_on_error:` alike; four-space
		// finally items are handled separately below.
		addNewlineBeforeTopLevelStepItem(),
		// Blank line before nested finally step items (exactly four spaces).
		addNewlineBeforeFinallyStepItem(),
		// Correcting first item newline under steps / finally / condition buckets
		replaceWith('steps:\n\n', (t) => t.replace('\n\n', '\n'), false),
		replaceWith('finally:\n\n', (t) => t.replace('\n\n', '\n'), false),
		replaceWith('  always:\n\n', (t) => t.replace('\n\n', '\n'), false),
		replaceWith('  on_success:\n\n', (t) => t.replace('\n\n', '\n'), false),
		replaceWith('  on_failure:\n\n', (t) => t.replace('\n\n', '\n'), false)
	);
}

// Utils

function enrichFollowUps(
	finallyDefinition: PipelineFinally | undefined
): Effect.Effect<EnrichedFollowUp[]> {
	const pending = getFinallyConditions().flatMap((condition) =>
		getFinallySteps(finallyDefinition, condition).map((step) => ({
			step: step as PipelineStep,
			condition
		}))
	);

	return Effect.forEach(
		pending,
		({ step, condition }) =>
			Effect.promise(async () => ({
				step: await enrichStep(step),
				condition
			})),
		{ concurrency: PIPELINE_ENRICH_CONCURRENCY }
	);
}

function getFinallyConditions(): PipelineFinallyCondition[] {
	return ['always', 'on_success', 'on_failure'];
}

function getFinallySteps(
	finallyDefinition: PipelineFinally | undefined,
	condition: PipelineFinallyCondition
) {
	if (!finallyDefinition) return [];
	if (Array.isArray(finallyDefinition)) {
		return condition === 'always' ? finallyDefinition : [];
	}
	return finallyDefinition[condition] ?? [];
}

async function enrichStep(step: PipelineStep): Promise<EnrichedStep> {
	if (step.use === 'debug') {
		return [step, {}];
	}

	try {
		const config = getConfigByTypeOrThrow(step.use);
		const data = await config.deserialize(step.with);
		return [step, data];
	} catch (e) {
		let error: Error | Enrich404Error | GenericRecord = {};
		if (e instanceof ClientResponseError) {
			if (e.status === 404) {
				error = new Enrich404Error();
			} else {
				error = new Error(e.message);
			}
		} else if (e instanceof Error) {
			error = e;
		} else {
			error = new Error(getExceptionMessage(e));
		}
		return [step, error];
	}
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
	return finallyStep;
}

function seedIdCounters(steps: PipelineStep[]) {
	const counters = new Map<string, number>();
	for (const step of steps) {
		const base = getIdBase(step);
		if (!base || !('id' in step)) continue;
		const suffix = getIdSuffix(step.id, base);
		if (suffix === undefined) continue;
		counters.set(base, Math.max(counters.get(base) ?? 0, suffix));
	}
	return counters;
}

function assignStepId(step: PipelineStep, counters: Map<string, number>) {
	const base = getIdBase(step);
	if (!base || !('id' in step)) return;

	const existingSuffix = getIdSuffix(step.id, base);
	if (existingSuffix !== undefined) {
		counters.set(base, Math.max(counters.get(base) ?? 0, existingSuffix));
		return;
	}

	const nextSuffix = (counters.get(base) ?? 0) + 1;
	counters.set(base, nextSuffix);
	step.id = `${base}-${nextSuffix.toString().padStart(4, '0')}`;
}

function getIdBase(step: PipelineStep): string | undefined {
	if (step.use === 'debug' || !('with' in step)) return undefined;
	try {
		const config = getConfigByTypeOrThrow(step.use);
		return slugify(config.makeId(step.with));
	} catch {
		// One bad makeId (e.g. template URL) must not blank the whole YAML preview.
		return slugify(step.use);
	}
}

function getIdSuffix(id: string, base: string) {
	const match = id.match(new RegExp(`^${_.escapeRegExp(base)}-(\\d+)$`));
	return match ? Number(match[1]) : undefined;
}

function addNewlineBefore(token: string, all = true) {
	return replaceWith(token, (token) => `\n${token}`, all);
}

/** Insert a blank line before any `  - ` item at the steps list indent only. */
function addNewlineBeforeTopLevelStepItem() {
	const pattern = /(?<=\n)(  - )/g;
	return (yaml: string) => yaml.replace(pattern, '\n$1');
}

/** Insert a blank line before `  always:` / `  on_success:` / `  on_failure:`. */
function addNewlineBeforeFinallyCondition(condition: PipelineFinallyCondition) {
	const pattern = new RegExp(`(?<=\\n)(  ${condition}:)`, 'g');
	return (yaml: string) => yaml.replace(pattern, '\n$1');
}

/** Insert a blank line before any `    - ` item under finally condition lists. */
function addNewlineBeforeFinallyStepItem() {
	const pattern = /(?<=\n)(    - )/g;
	return (yaml: string) => yaml.replace(pattern, '\n$1');
}

function replaceWith(token: string, transform: (token: string) => string, all = true) {
	if (all) {
		return String.replaceAll(token, transform(token));
	} else {
		return String.replace(token, transform(token));
	}
}
