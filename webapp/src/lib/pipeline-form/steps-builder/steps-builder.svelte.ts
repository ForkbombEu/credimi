// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Renderable } from '$lib/renderable';

import { confirm } from '$lib/layout/global-confirm.svelte';
import { StateManager } from '$lib/state-manager/state-manager';
import { isError } from 'effect/Predicate';
import { cloneDeep } from 'lodash';

import type { GenericRecord } from '@/utils/types';

import { m } from '@/i18n';

import type { PipelineStep, PipelineStepByType } from '../../pipeline/types';
import type { SelectedVersion } from '../execution-target/types.js';
import type { WalletActionStepData } from '../steps/wallet-action/types.js';
import type { EnrichedStep } from './types';

import { showPipelineFormError } from '../errors.js';
import { resolveExecutionTarget } from '../execution-target/index.js';
import * as pipelinestep from '../steps';
import { walletActionStepConfig } from '../steps/wallet-action/index.js';
import { getBulkWalletVersionContext } from './_partials/bulk-wallet-version-context.js';
import { getStepData, isStepEditable } from './_partials/utils.js';
import { isExecutionTargetLocked } from './execution-target-lock.js';
import { InlineManualEditor } from './inline-manual-editor.svelte.js';
import Component from './steps-builder.svelte';

//

type Props = {
	steps: EnrichedStep[];
	yamlPreview: () => string;
	isSavedManualPipeline?: boolean;
};

type BuilderMode =
	| { id: 'idle' }
	| {
			id: 'form';
			intent: pipelinestep.FormIntent;
			stepIndex?: number;
			config: pipelinestep.AnyConfig;
			form: pipelinestep.Form;
	  }
	| { id: 'manual'; editor: InlineManualEditor };

type State = {
	steps: EnrichedStep[];
	mode: BuilderMode;
	manualLocked: boolean;
};

export class StepsBuilder implements Renderable<StepsBuilder> {
	readonly Component = Component;

	private state = $state<State>({
		steps: [],
		mode: { id: 'idle' },
		manualLocked: false
	});

	private stateManager = new StateManager(
		() => this.state,
		(state) => (this.state = state)
	);

	private formEffectCleanup: (() => void) | null = null;

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

	executionTarget = $derived(resolveExecutionTarget(this.state.steps));

	readonly yamlPreview = $derived.by(() => {
		try {
			return this.props.yamlPreview();
		} catch (e) {
			showPipelineFormError(e);
			return '';
		}
	});

	get isManualMode() {
		return this.state.mode.id === 'manual';
	}

	get isFormMode() {
		return this.state.mode.id === 'form';
	}

	get isManualLocked() {
		return this.state.manualLocked;
	}

	get isSavedManualPipeline() {
		return this.props.isSavedManualPipeline === true;
	}

	undo() {
		this.stateManager.undo();
	}

	redo() {
		this.stateManager.redo();
	}

	// Core functionality

	initAddStep(type: string) {
		if (this.state.mode.id === 'form') {
			this.exitFormState();
		}
		const config = pipelinestep.getConfigByType(type as PipelineStep['use']);
		if (!config) return;
		this.openForm('add', config, {});
	}

	initEditStep(index: number) {
		if (this.state.mode.id === 'form') {
			this.exitFormState();
		}
		const step = this.state.steps[index];
		if (!step || !isStepEditable(step)) return;
		const config = pipelinestep.getConfigByType(step[0].use);
		const data = getStepData(step);
		if (!config || !data) return;
		this.openForm('edit', config, { initial: data, stepIndex: index });
	}

	private openForm(
		intent: pipelinestep.FormIntent,
		config: pipelinestep.AnyConfig,
		opts: { initial?: GenericRecord; stepIndex?: number }
	) {
		this.stateManager.run((state) => {
			const effectCleanup = $effect.root(() => {
				let form: pipelinestep.Form;
				try {
					form = config.initForm({
						intent,
						initial: opts.initial as never,
						getExecutionTarget: () => this.executionTarget,
						isExecutionTargetLocked: () =>
							isExecutionTargetLocked({
								intent,
								steps: this.state.steps,
								target: this.executionTarget
							})
					});
				} catch (e) {
					showPipelineFormError(e);
					return;
				}
				form.onSubmit((formData) => {
					try {
						this.stateManager.run((inner) => {
							if (inner.mode.id !== 'form') return;

							if (inner.mode.intent === 'add') {
								const step: PipelineStep = {
									use: config.use as never,
									id: '',
									continue_on_error: false,
									with: config.serialize(formData)
								};
								inner.steps.push([step, formData as GenericRecord]);
							} else {
								const editIndex = inner.mode.stepIndex;
								if (editIndex === undefined) return;
								const tuple = inner.steps[editIndex];
								if (!tuple || tuple[0].use === 'debug') return;
								tuple[0].with = config.serialize(formData);
								tuple[1] = formData as GenericRecord;
							}

							inner.mode = { id: 'idle' };
						});
						this.disposeFormEffect();
					} catch (e) {
						showPipelineFormError(e);
					}
				});
				state.mode = {
					id: 'form',
					intent,
					stepIndex: opts.stepIndex,
					config,
					form
				};
			});
			this.formEffectCleanup = effectCleanup;
		});
	}

	addDebugStep() {
		this.stateManager.run((state) => {
			state.steps.push([{ use: 'debug' }, {}]);
		});
	}

	deleteStep(index: number) {
		if (this.isFormMode) return;
		this.stateManager.run((state) => {
			state.steps.splice(index, 1);
		});
	}

	cloneStep(index: number) {
		if (this.isFormMode) return;
		this.stateManager.run((state) => {
			const source = state.steps[index];
			if (!source) return;
			const [pipelineStep, formData] = cloneDeep(source);
			if (pipelineStep.use !== 'debug' && 'id' in pipelineStep) {
				pipelineStep.id = '';
			}
			state.steps.splice(index + 1, 0, [pipelineStep, formData]);
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
			if (state.mode.id !== 'form') return;
			state.mode = { id: 'idle' };
		});
		this.disposeFormEffect();
	}

	enterManualMode(initialYaml: string, options?: { locked?: boolean }) {
		if (this.state.mode.id === 'form') {
			this.exitFormState();
		}
		const editor = new InlineManualEditor(initialYaml);
		this.stateManager.run((state) => {
			state.mode = { id: 'manual', editor };
			state.manualLocked = options?.locked ?? false;
		});
		void editor.validateNow();
	}

	async exitManualMode(): Promise<boolean> {
		if (this.state.mode.id !== 'manual') return true;
		if (this.state.manualLocked) return true;

		const { editor } = this.state.mode;
		if (editor.isDirty) {
			const confirmed = await confirm({
				message:
					m.discard_manual_yaml_changes() +
					'\n' +
					m.Are_you_sure_you_want_to_exit_the_form(),
				destructive: true
			});
			if (!confirmed) return false;
		}
		editor.dispose();
		this.stateManager.run((state) => {
			state.mode = { id: 'idle' };
			state.manualLocked = false;
		});
		return true;
	}

	private disposeFormEffect() {
		this.formEffectCleanup?.();
		this.formEffectCleanup = null;
	}

	// Ordering

	shiftStep(index: number, change: number) {
		if (this.isFormMode) return;
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

	//

	applyBulkWalletVersion(version: SelectedVersion) {
		const ctx = getBulkWalletVersionContext(this.state.steps);
		if (!ctx) return;
		this.stateManager.run((state) => {
			state.steps = this.syncMobileStepVersions(state.steps, ctx.wallet.id, version);
		});
	}

	private syncMobileStepVersions(
		steps: EnrichedStep[],
		walletId: string,
		version: SelectedVersion
	): EnrichedStep[] {
		return steps.map((tuple) => {
			const [raw, data] = tuple;
			if (raw.use !== 'mobile-automation') return tuple;
			if (isError(data)) return tuple;

			const stepData = data as unknown as WalletActionStepData;
			if (stepData.wallet.id !== walletId) return tuple;

			const updated: WalletActionStepData = { ...stepData, version };
			const nextRaw = {
				...raw,
				with: walletActionStepConfig.serialize(updated)
			} as PipelineStepByType<'mobile-automation'>;

			return [nextRaw, updated as GenericRecord] as EnrichedStep;
		});
	}
}
