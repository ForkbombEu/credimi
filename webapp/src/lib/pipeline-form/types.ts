// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';
import type { Renderable } from '$lib/renderable';
import type { Snippet } from 'svelte';
import type { Simplify } from 'type-fest';

import type * as t from './types.generated.js';

/* Core types */

export type Pipeline = t.HttpsGithubComForkbombeuCredimiPkgWorkflowenginePipelineWorkflowDefinition;

export type ActivityOptions = t.ActivityOptions;

export type PipelineStep = NonNullable<Pipeline['steps']>[number];

export type PipelineStepType = PipelineStep['use'];

/* Pipeline Step Config types */

export interface PipelineStepConfig<
	ID extends PipelineStepType = never,
	Serialized = unknown,
	Deserialized = unknown
> {
	id: ID;
	serialize: (step: Deserialized) => Serialized;
	deserialize: (step: Serialized) => Promise<Deserialized>;
	display: EntityData;
	initForm: () => PipelineStepDataForm<Deserialized>;
	snippet?: Snippet<[{ data: Deserialized; display: EntityData }]>;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface PipelineStepDataForm<Deserialized = unknown, T = any> extends Renderable<T> {
	onSubmit: (handler: (step: Deserialized) => void) => void;
}

/* Typed helpers */

export type PipelineStepWithId = Extract<PipelineStep, { id: string }>;

export type PipelineStepByType<T extends PipelineStepType> = Simplify<
	Extract<PipelineStep, { use: T }>
>;

export type PipelineStepData<Step extends PipelineStep> = Step extends { with: infer W }
	? W
	: never;

export type TypedPipelineStepConfig<T extends PipelineStepType, Deserialized> = Simplify<
	PipelineStepConfig<T, PipelineStepData<PipelineStepByType<T>>, Deserialized>
>;

/* Utilities */

export abstract class BasePipelineStepDataForm<Deserialized, T>
	implements PipelineStepDataForm<Deserialized, T>
{
	abstract Component: Renderable<T>['Component'];

	protected handleSubmit: (step: Deserialized) => void = () => {};

	onSubmit(handler: (data: Deserialized) => void) {
		this.handleSubmit = handler;
	}
}
