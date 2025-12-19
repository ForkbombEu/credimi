// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityUIData } from '$lib/globals/entities.js';
import type { Renderable } from '$lib/renderable';
import type { Snippet } from 'svelte';
import type { Simplify } from 'type-fest';

import type * as t from './types.generated.js';

/* Core types */

export type Pipeline = t.HttpsGithubComForkbombeuCredimiPkgWorkflowenginePipelineWorkflowDefinition;

export type ActivityOptions = t.ActivityOptions;

/* Pipeline Step types (derived) */

export type PipelineStep = NonNullable<Pipeline['steps']>[number];

export interface PipelineStepConfig<ID = string, Serialized = unknown, Deserialized = unknown> {
	id: ID;
	serialize: (step: Deserialized) => Serialized;
	deserialize: (step: Serialized) => Promise<Deserialized>;
	display: EntityUIData;
	initForm: () => PipelineStepDataForm;
	snippet?: Snippet<[{ data: Deserialized; display: EntityUIData }]>;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface PipelineStepDataForm<T = any> extends Renderable<T> {
	onSubmit: (handler: (step: PipelineStep) => void) => void;
}

/* Typed helpers */

export type PipelineStepType = PipelineStep['use'];

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

export abstract class BasePipelineStepDataForm<T> implements PipelineStepDataForm<T> {
	abstract Component: Renderable<T>['Component'];

	handleSubmit: (step: PipelineStep) => void = () => {};

	onSubmit(handler: (step: PipelineStep) => void) {
		this.handleSubmit = handler;
	}
}
