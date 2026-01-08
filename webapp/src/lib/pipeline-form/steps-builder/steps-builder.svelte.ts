// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Renderable } from '$lib/renderable';
import { StateManager } from '$lib/state-manager/state-manager';
import { nanoid } from 'nanoid';
import type {
	AnyPipelineStepConfig,
	PipelineStep,
	PipelineStepDataForm,
	PipelineStepWithId
} from '../types';
import Component from './steps-builder.svelte';

//

export type EnrichedStep = [PipelineStep, Record<string, unknown>];

type Props = {
	configs: AnyPipelineStepConfig[];
	steps: EnrichedStep[];
	yamlPreview: () => string;
};

type StepsBuilderState = {
	steps: EnrichedStep[];
	state: { id: 'idle' } | { id: 'form'; form: PipelineStepDataForm };
};

export class StepsBuilder implements Renderable<StepsBuilder> {
	readonly Component = Component;

	private _state = $state<StepsBuilderState>({
		steps: [],
		state: { id: 'idle' }
	});
	private stateManager = new StateManager(this._state);

	constructor(private props: Props) {
		this._state.steps = props.steps;
	}

	// Shortcuts

	get state() {
		return this._state.state;
	}

	get steps() {
		return this._state.steps;
	}

	get yamlPreview() {
		return this.props.yamlPreview();
	}

	get configs() {
		return this.props.configs;
	}

	undo() {
		this.stateManager.undo();
	}

	redo() {
		this.stateManager.redo();
	}

	// Core functionality

	initAddStep(type: string) {
		this.stateManager.run((data) => {
			const config = this.props.configs.find((c) => c.id === type);
			if (!config) return;

			const effectCleanup = $effect.root(() => {
				const form = config.initForm();
				form.onSubmit((formData) => {
					const step: PipelineStepWithId = {
						id: nanoid(5),
						use: config.id,
						with: config.serialize(formData) as Record<string, unknown>,
						continue_on_error: false
					};
					this.stateManager.run((data) => {
						data.steps.push([step, formData]);
						data.state = { id: 'idle' };
						effectCleanup();
					});
				});
				data.state = { id: 'form', form };
			});
		});
	}

	addDebugStep() {
		this.stateManager.run((data) => {
			data.steps.push([{ use: 'debug' }, {}]);
		});
	}

	deleteStep(index: number) {
		this.stateManager.run((data) => {
			data.steps.splice(index, 1);
		});
	}

	setContinueOnError(index: number, continueOnError: boolean) {
		this.stateManager.run((data) => {
			const step = data.steps[index];
			if (!step || step[0].use == 'debug') return;
			step[0].continue_on_error = continueOnError;
		});
	}

	exitFormState() {
		if (!(this.state.id == 'form')) return;
		this.stateManager.run((data) => {
			data.state = { id: 'idle' };
		});
	}

	// Ordering

	shiftStep(index: number, change: number) {
		this.stateManager.run((data) => {
			const indices = this.calculateShiftIndices(index, change);
			if (!indices) return;
			const [movedItem] = data.steps.splice(indices.index, 1);
			data.steps.splice(indices.newIndex, 0, movedItem);
		});
	}

	canShiftStep(index: number, change: number) {
		return this.calculateShiftIndices(index, change) !== null;
	}

	private calculateShiftIndices(index: number, change: number) {
		const newIndex = index + change;
		if (newIndex < 0 || newIndex >= this.steps.length || newIndex === index) return null;
		return { index, newIndex };
	}

	// Utilities

	// TODO: Convert to "hasChanges"
	hasSteps() {
		return this.steps.length > 0;
	}

	getConfig(type: string) {
		return this.props.configs.find((c) => c.id === type);
	}
}
