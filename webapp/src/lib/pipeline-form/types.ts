// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityUIData } from '$lib/globals/entities.js';
import type { Renderable } from '$lib/renderable';
import type { Snippet } from 'svelte';

import type * as t from './types.generated.js';

/* Core types */

export type Pipeline = t.HttpsGithubComForkbombeuCredimiPkgWorkflowenginePipelineWorkflowDefinition;

export type ActivityOptions = t.ActivityOptions;

/* Pipeline Step types (derived) */

export type PipelineStep = NonNullable<Pipeline['steps']>[number];

// export type PipelineStepType = PipelineStep['use'];

// export type PipelineStepByType<T extends PipelineStepType> = Simplify<
// 	Extract<PipelineStep, { use: T }>
// >;

export interface PipelineStepConfig<ID = string, YamlStep = unknown, UIStepData = unknown> {
	id: ID;
	serialize: (step: UIStepData) => YamlStep;
	deserialize: (step: YamlStep) => Promise<UIStepData>;
	display: EntityUIData;
	initForm: (onSubmit: (step: PipelineStep) => void) => PipelineDataForm;
	snippet?: Snippet<[{ data: UIStepData; display: EntityUIData }]>;
}

export interface PipelineDataForm extends Renderable {
	onSubmit: (handler: (step: PipelineStep) => void) => void;
}

// interface WithComponentBase<T> {
// 	readonly Component: Component<{ self: T }>;
//   }

//   interface WithComponentTyped<T extends WithComponentTyped<T>>
// 	extends WithComponentBase<T> {}
