// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Simplify } from 'type-fest';

import type { PipelinesResponse } from '@/pocketbase/types/index.generated.js';

import type { EnrichedStep } from './steps-builder/types';
import type * as t from './types.generated.js';

// Core types

export type Pipeline = t.HttpsGithubComForkbombeuCredimiPkgWorkflowenginePipelineWorkflowDefinition;

export type ActivityOptions = t.ActivityOptions;

export type PipelineStep = NonNullable<Pipeline['steps']>[number];

export type PipelineStepType = PipelineStep['use'];

export type PipelineStepByType<T extends PipelineStepType> = Simplify<
	Extract<PipelineStep, { use: T }>
>;

export type PipelineStepWithId = Extract<PipelineStep, { id: string }>;

export type PipelineStepData<Step extends PipelineStep> = Step extends { with: infer W }
	? W
	: never;

// Other

export interface EnrichedPipeline {
	record: PipelinesResponse;
	activity_options?: ActivityOptions;
	steps: EnrichedStep[];
}
