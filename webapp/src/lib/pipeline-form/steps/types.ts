// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';
import type {
	PipelineStep,
	PipelineStepByType,
	PipelineStepData,
	PipelineStepType
} from '$lib/pipeline/types';
import type { Renderable } from '$lib/renderable';
import type { Component } from 'svelte';
import type { Simplify } from 'type-fest';

// Pipeline Step Config

export type FormIntent = 'add' | 'edit';

export type InitFormOptions<Deserialized = unknown> = {
	intent?: FormIntent;
	initial?: Deserialized;
};

export interface Config<ID extends string = string, Serialized = unknown, Deserialized = unknown> {
	use: ID;
	serialize: (step: Deserialized) => Serialized;
	deserialize: (step: Serialized) => Promise<Deserialized>;
	display: EntityData;
	initForm: (opts?: InitFormOptions<Deserialized>) => Form<Deserialized>;
	cardData: (data: Deserialized) => CardData;
	CardDetailsComponent?: Component<CardDetailsComponentProps<Deserialized>>;
	makeId: (data: Serialized) => string;
	linkProcedure?: (serialized: Serialized, previousSteps: PipelineStep[]) => void;
}

export type CardDetailsComponentProps<Deserialized = unknown> = {
	data: Deserialized;
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface Form<Deserialized = unknown, T = any> extends Renderable<T> {
	readonly intent: FormIntent;
	onSubmit: (handler: (step: Deserialized) => void) => void;
	canSave(): boolean;
	getSubmitData(): Deserialized | undefined;
	commit(data?: Deserialized): void;
}

export interface CardData {
	title: string;
	copyText?: string;
	avatar?: string;
	meta?: Record<string, unknown>;
	publicUrl?: string;
	beforeTitle?: string;
}

// Utilities

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type AnyConfig = Config<string, any, any>;

export type TypedConfig<T extends PipelineStepType, Deserialized> = Simplify<
	Config<T, PipelineStepData<PipelineStepByType<T>>, Deserialized>
>;

export abstract class BaseForm<Deserialized, T> implements Form<Deserialized, T> {
	abstract Component: Renderable<T>['Component'];

	readonly intent: FormIntent;
	protected handleSubmit: (step: Deserialized) => void = () => {};

	constructor(opts?: InitFormOptions<Deserialized>) {
		this.intent = opts?.intent ?? 'add';
		if (opts?.initial !== undefined) {
			this.applyInitial(opts.initial);
		}
	}

	protected applyInitial(_initial: Deserialized): void {
		void _initial;
		// overridden per form
	}

	onSubmit(handler: (data: Deserialized) => void) {
		this.handleSubmit = handler;
	}

	commit(data?: Deserialized) {
		const payload = data ?? this.getSubmitData();
		if (payload !== undefined) {
			this.handleSubmit(payload);
		}
	}

	abstract canSave(): boolean;
	abstract getSubmitData(): Deserialized | undefined;
}
