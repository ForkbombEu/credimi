// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityUIData } from '$lib/globals/entities.js';
import type { Component, Snippet } from 'svelte';
import type { SetFieldType, Simplify } from 'type-fest';

import type * as t from './types.generated.js';

//

export type Pipeline = t.HttpsGithubComForkbombeuCredimiPkgWorkflowenginePipelineWorkflowDefinition;

export type ActivityOptions = t.ActivityOptions;

export type PipelineStep = NonNullable<Pipeline['steps']>[number];

export type BasePipelineStep = Simplify<
	SetFieldType<
		SetFieldType<Omit<PipelineStep, 'with'>, 'metadata', Record<string, unknown>>,
		'use',
		string
	>
>;

export type PipelineStepType = PipelineStep['use'];

export type PipelineStepByType<T extends PipelineStepType> = Simplify<
	Extract<PipelineStep, { use: T }>
>;

export interface Config<ID = string, YamlStep = unknown, UIStepData = unknown> {
	id: ID;
	serialize: (step: UIStepData) => YamlStep;
	deserialize: (step: YamlStep) => Promise<UIStepData>;
	display: EntityUIData;
	form?: UIStepDataForm;
	snippet?: Snippet<[{ data: UIStepData; display: EntityUIData }]>;
}

export interface WithComponent {
	readonly Component: Component<{ readonly self: WithComponent }>;
}

export interface UIStepDataForm extends WithComponent {
	onSubmit: (handler: (step: unknown) => void) => void;
}

interface StepsForm {
	configs: Config[];
	state: 'idle' | 'form';
	selectStep: (id: string) => void;
	handleStepFormSubmit: (step: unknown) => void;
	exitStepForm: () => void;
	shiftStep: (id: string, direction: 'up' | 'down') => void;
	deleteStep: (id: string) => void;
}

// interface WithComponentBase<T> {
// 	readonly Component: Component<{ self: T }>;
//   }

//   interface WithComponentTyped<T extends WithComponentTyped<T>>
// 	extends WithComponentBase<T> {}
