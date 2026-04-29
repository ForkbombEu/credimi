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

import type { RuntimeOptions } from './runtime-options-form/runtime-options-form.svelte.js';

import { type Pipeline, type PipelineStep } from '../pipeline/types';
import { getConfigByType } from './steps';
import { Enrich404Error, type EnrichedStep } from './steps-builder/types';

/* Fetching pipeline */

export interface EnrichedPipeline {
	record: PipelinesResponse;
	runtime?: RuntimeOptions;
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
				const config = getConfigByType(step.use);
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
		runtime: yaml.runtime,
		steps: enrichedSteps
	};
}

/* YAML processing */

export function createPipelineYaml(
	name: string,
	steps: PipelineStep[],
	runtime: RuntimeOptions
): string {
	// Cloning because editing ids and linking steps modify the original steps array
	const clonedSteps = _.cloneDeep(steps);

	const processedSteps = clonedSteps.map((step, index) => {
		const config = getConfigByType(step.use);
		if ('id' in step) {
			step.id = `${slugify(config.makeId(step.with))}-${(index + 1).toString().padStart(4, '0')}`;
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
