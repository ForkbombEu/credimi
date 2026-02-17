// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';
import type { PipelineStepByType, PipelineStepData, PipelineStepType } from '$lib/pipeline/types';
import type { Renderable } from '$lib/renderable';
import type { Component } from 'svelte';
import type { Simplify } from 'type-fest';

// Pipeline Step Config

export interface Config<ID extends string = string, Serialized = unknown, Deserialized = unknown> {
	use: ID;
	serialize: (step: Deserialized) => Serialized;
	deserialize: (step: Serialized) => Promise<Deserialized>;
	display: EntityData;
	initForm: () => Form<Deserialized>;
	cardData: (data: Deserialized) => CardData;
	CardDetailsComponent?: Component<{ data: Deserialized }>;
	makeId: (data: Serialized) => string;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface Form<Deserialized = unknown, T = any> extends Renderable<T> {
	onSubmit: (handler: (step: Deserialized) => void) => void;
}

export interface CardData {
	title: string;
	copyText?: string;
	avatar?: string;
	meta?: Record<string, unknown>;
}

// Utilities

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type AnyConfig = Config<string, any, any>;

export type TypedConfig<T extends PipelineStepType, Deserialized> = Simplify<
	Config<T, PipelineStepData<PipelineStepByType<T>>, Deserialized>
>;

export abstract class BaseForm<Deserialized, T> implements Form<Deserialized, T> {
	abstract Component: Renderable<T>['Component'];

	protected handleSubmit: (step: Deserialized) => void = () => {};

	onSubmit(handler: (data: Deserialized) => void) {
		this.handleSubmit = handler;
	}
}
