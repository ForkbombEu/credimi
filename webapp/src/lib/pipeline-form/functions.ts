// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pipe, String } from 'effect';
import * as _ from 'lodash';
import { ClientResponseError } from 'pocketbase';
import slugify from 'slugify';
import { parse, stringify } from 'yaml';

import type { PipelinesResponse } from '@/pocketbase/types';
import type { GenericRecord } from '@/utils/types.js';

import { pb } from '@/pocketbase';
import { getExceptionMessage } from '@/utils/errors.js';

import {
	type ActivityOptions,
	type Pipeline,
	type PipelineStep,
	type PipelineStepType
} from '../pipeline/types';
import { configs } from './steps';
import { Enrich404Error, type EnrichedStep } from './steps-builder/types';

/* Fetching pipeline */

export interface EnrichedPipeline {
	record: PipelinesResponse;
	activity_options?: ActivityOptions;
	steps: EnrichedStep[];
}

export async function getEnrichedPipeline(
	id: string,
	options = { fetch }
): Promise<EnrichedPipeline> {
	const record = await pb.collection('pipelines').getOne(id, { fetch: options.fetch });

	const yaml = parse(record.yaml) as Pipeline;
	const steps = yaml.steps ?? [];

	const enrichedSteps: EnrichedStep[] = [];
	for (const step of steps) {
		if (step.use === 'debug') {
			enrichedSteps.push([step, {}]);
		} else {
			try {
				const config = configs.find((c) => c.use === step.use);
				if (!config) throw new Error(`Unknown step type: ${step.use}`);
				const data = await config.deserialize(step.with);
				enrichedSteps.push([step, data]);
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
				enrichedSteps.push([step, error]);
			}
		}
	}

	return {
		record,
		activity_options: yaml.runtime?.temporal?.activity_options,
		steps: enrichedSteps
	};
}

/* YAML processing */

export function createPipelineYaml(
	name: string,
	steps: PipelineStep[],
	activity_options: ActivityOptions
): string {
	// Cloning because addProgressiveStepIds and linkIds modify the original steps array
	const processedSteps = pipe(_.cloneDeep(steps), generateIds, linkIds);

	const pipeline: Pipeline = {
		name,
		runtime: {
			temporal: {
				activity_options
			}
		},
		steps: processedSteps
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

//

function generateIds(steps: PipelineStep[]): PipelineStep[] {
	for (const [index, step] of steps.entries()) {
		if (!('id' in step)) continue;

		const config = configs.find((c) => c.use === step.use);
		if (!config) throw new Error(`Unknown step type: ${step.use}`);

		step.id = `${slugify(config.makeId(step.with))}-${(index + 1).toString().padStart(4, '0')}`;
	}
	return steps;
}

//

const linkableSteps: PipelineStepType[] = [
	'conformance-check',
	'credential-offer',
	'use-case-verification-deeplink',
	'custom-check'
];

function linkIds(steps: PipelineStep[]): PipelineStep[] {
	for (const [index, step] of steps.entries()) {
		if (!(step.use === 'mobile-automation')) continue;
		if (!step.with.parameters) continue;
		if (!('deeplink' in step.with.parameters)) continue;

		const previousStep = steps
			.slice(0, index)
			.toReversed()
			.filter((s) => linkableSteps.includes(s.use))
			.at(0);

		if (!previousStep || !('id' in previousStep)) continue;

		let deeplinkPath = '.outputs';
		if (previousStep.use === 'conformance-check') {
			deeplinkPath += '.deeplink';
		}

		step.with.parameters.deeplink = '${{' + previousStep.id + deeplinkPath + '}}';
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
