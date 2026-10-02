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
import { Effect } from 'effect';
import { ClientResponseError } from 'pocketbase';
import { parse } from 'yaml';

import type { PipelinesResponse } from '@/pocketbase/types';
import type { GenericRecord } from '@/utils/types.js';

import { pb } from '@/pocketbase';
import { getExceptionMessage } from '@/utils/errors.js';

import type { RuntimeOptions } from './runtime-options-form/runtime-options-form.svelte.js';

import { getFinallyConditions, getFinallySteps } from './pipeline-finally.js';
import { getConfigByTypeOrThrow } from './steps';

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

/** Empty enriched shell (record only) for loading UI / enrichment fallbacks. */
export function minimalEnrichedPipeline(record: PipelinesResponse): EnrichedPipeline {
	return { record, steps: [], runtime: undefined };
}

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
