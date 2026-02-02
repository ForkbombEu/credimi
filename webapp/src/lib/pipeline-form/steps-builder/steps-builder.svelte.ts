// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Renderable } from '$lib/renderable';

import { StateManager } from '$lib/state-manager/state-manager';

import type { PipelineStep } from '../types';
import type { EnrichedStep } from './types';

import * as pipelinestep from '../steps';
import Component from './steps-builder.svelte';

//

type Props = {
	steps: EnrichedStep[];
	yamlPreview: () => string;
};

type State = {
	steps: EnrichedStep[];
	mode: { id: 'idle' } | { id: 'form'; form: pipelinestep.Form };
};

export class StepsBuilder implements Renderable<StepsBuilder> {
	readonly Component = Component;

	private state = $state<State>({
		steps: [],
		mode: { id: 'idle' }
	});

	private stateManager = new StateManager(
		() => this.state,
		(state) => (this.state = state)
	);

	constructor(private props: Props) {
		this.state.steps = props.steps;
	}

	// Shortcuts

	get mode() {
		return this.state.mode;
	}

	get steps() {
		return this.state.steps;
	}

	get yamlPreview() {
		return this.props.yamlPreview();
	}

	undo() {
		this.stateManager.undo();
	}

	redo() {
		this.stateManager.redo();
	}

	// Core functionality

	initAddStep(type: string) {
		this.stateManager.run((state) => {
			const config = pipelinestep.configs.find((c) => c.use === type);
			if (!config) return;

			const effectCleanup = $effect.root(() => {
				const form = config.initForm();
				form.onSubmit((formData) => {
					this.stateManager.run((state) => {
						const step: PipelineStep = {
							use: config.use as never,
							id: '', // will be written later
							continue_on_error: true,
							with: config.serialize(formData)
						};
						state.steps.push([step, formData]);
						state.mode = { id: 'idle' };
						effectCleanup();
					});
				});
				state.mode = { id: 'form', form };
			});
		});
	}

	addDebugStep() {
		this.stateManager.run((state) => {
			state.steps.push([{ use: 'debug' }, {}]);
		});
	}

	deleteStep(index: number) {
		this.stateManager.run((state) => {
			state.steps.splice(index, 1);
		});
	}

	setContinueOnError(index: number, continueOnError: boolean) {
		this.stateManager.run((state) => {
			const step = state.steps[index];
			if (!step || step[0].use == 'debug') return;
			step[0].continue_on_error = continueOnError;
		});
	}

	exitFormState() {
		this.stateManager.run((state) => {
			if (!(state.mode.id == 'form')) return;
			state.mode = { id: 'idle' };
		});
	}

	// Ordering

	shiftStep(index: number, change: number) {
		this.stateManager.run((state) => {
			const indices = this.calculateShiftIndices(state, index, change);
			if (!indices) return;
			const [movedItem] = state.steps.splice(indices.index, 1);
			state.steps.splice(indices.newIndex, 0, movedItem);
		});
	}

	canShiftStep(index: number, change: number) {
		return this.calculateShiftIndices(this.state, index, change) !== null;
	}

	private calculateShiftIndices(state: State, index: number, change: number) {
		const newIndex = index + change;
		if (newIndex < 0 || newIndex >= state.steps.length || newIndex === index) return null;
		return { index, newIndex };
	}
}
