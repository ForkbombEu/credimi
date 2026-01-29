// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Renderable } from '$lib/renderable';

import { StateManager } from '$lib/state-manager/state-manager';
import { onDestroy } from 'svelte';

import type { PipelineStep } from '../types';
import type { EnrichedStep } from './types';

import * as pipelinestep from '../steps';
import { walletActionStepFormState } from '../steps/wallet-action/wallet-action-step-form.svelte.js';
import Component from './steps-builder.svelte';

//

type Props = {
	steps: EnrichedStep[];
	yamlPreview: () => string;
};

type State = {
	steps: EnrichedStep[];
	state: { id: 'idle' } | { id: 'form'; form: pipelinestep.DataForm };
};

export class StepsBuilder implements Renderable<StepsBuilder> {
	readonly Component = Component;

	private _state = $state<State>({
		steps: [],
		state: { id: 'idle' }
	});
	private stateManager = new StateManager(
		() => this._state,
		(state) => (this._state = state)
	);

	constructor(private props: Props) {
		this._state.steps = props.steps;
		
		// Initialize runner type constraint from first step if it exists
		if (props.steps.length > 0) {
			const firstStep = props.steps[0];
			if (firstStep[0].use === 'mobile-automation') {
				// Check if the runner is global or specific
				// We need to check the runner data from the deserialized form data
				const firstStepData = firstStep[1] as any;
				if (firstStepData && firstStepData.runner) {
					const isGlobalRunner = firstStepData.runner.published;
					walletActionStepFormState.firstStepRunnerType = isGlobalRunner ? 'global' : 'specific';
				}
			}
		}

		onDestroy(() => {
			walletActionStepFormState.lastSelectedWallet = undefined;
			walletActionStepFormState.firstStepRunnerType = undefined;
			walletActionStepFormState.isFirstStep = false;
		});
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

	undo() {
		this.stateManager.undo();
	}

	redo() {
		this.stateManager.redo();
	}

	// Core functionality

	initAddStep(type: string) {
		const config = pipelinestep.configs.find((c) => c.use === type);
		if (!config) return;

		this.stateManager.run((data) => {
			const config = pipelinestep.configs.find((c) => c.use === type);
			if (!config) return;

			const effectCleanup = $effect.root(() => {
				// Set whether this is the first step for wallet-action forms
				if (config.use === 'mobile-automation') {
					walletActionStepFormState.isFirstStep = data.steps.length === 0;
				}
				
				const form = config.initForm();
				form.onSubmit((formData) => {
					const step: PipelineStep = {
						use: config.use as never,
						id: '', // will be written later
						continue_on_error: true,
						with: config.serialize(formData)
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
			
			// If all steps are deleted, reset the runner type constraint
			if (data.steps.length === 0) {
				walletActionStepFormState.firstStepRunnerType = undefined;
			}
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
}
