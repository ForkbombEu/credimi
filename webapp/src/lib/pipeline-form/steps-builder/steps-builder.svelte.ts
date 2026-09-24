// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineStep, PipelineStepByType } from '$lib/pipeline/types';
import type { Renderable } from '$lib/renderable';
import type { SelectedVersion } from '$pipeline-form/execution-target/types.js';
import type { EnrichedFollowUp } from '$pipeline-form/functions.js';
import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
import type { WalletActionStepData } from '$pipeline-form/steps/wallet-action/types.js';

import { confirm } from '$lib/layout/global-confirm.svelte';
import { StateManager } from '$lib/state-manager/state-manager';
import { showPipelineFormError } from '$pipeline-form/errors.js';
import { resolveExecutionTarget } from '$pipeline-form/execution-target/index.js';
import * as pipelinestep from '$pipeline-form/steps';
import { walletActionStepConfig } from '$pipeline-form/steps/wallet-action/index.js';
import { isError } from 'effect/Predicate';
import { cloneDeep } from 'lodash';

import type { GenericRecord } from '@/utils/types';

import { m } from '@/i18n';

import {
	getBulkWalletVersionContext,
	getStepData,
	isChangeWalletVersionAvailable,
	isStepEditable
} from './_partials/index.js';
import { isExecutionTargetLocked } from './execution-target-lock.js';
import { InlineManualEditor } from './inline-manual-editor.svelte.js';
import Component from './steps-builder.svelte';

//

type Props = {
	steps: EnrichedStep[];
	followUps?: EnrichedFollowUp[];
	yamlPreview: () => string;
	isSavedManualPipeline?: boolean;
};

type FormSection = 'steps' | 'follow-ups';

type BuilderMode =
	| { id: 'idle' }
	| {
			id: 'form';
			intent: pipelinestep.FormIntent;
			stepIndex?: number;
			section?: FormSection;
			config: pipelinestep.AnyConfig;
			form: pipelinestep.Form;
	  }
	| { id: 'manual'; editor: InlineManualEditor };

type State = {
	steps: EnrichedStep[];
	followUps: EnrichedFollowUp[];
	mode: BuilderMode;
	manualLocked: boolean;
};

type CreatedCardRef = {
	section: 'steps' | 'follow-ups';
	index: number;
	token: number;
};

export class StepsBuilder implements Renderable<StepsBuilder> {
	readonly Component = Component;

	private state = $state<State>({
		steps: [],
		followUps: [],
		mode: { id: 'idle' },
		manualLocked: false
	});

	private stateManager = new StateManager(
		() => this.state,
		(state) => (this.state = state)
	);

	private formEffectCleanup: (() => void) | null = null;

	changeWalletVersionDialogOpen = $state(false);

	/** View-bound scroll-follow hooks (reveal / edit-focus). Not product state. */
	#composerScroll: {
		onRevealStep?: (index: number) => void;
		onEditFocus?: (stepIndex: number) => void;
	} = {};

	createdCard = $state<CreatedCardRef | null>(null);

	private createdCardToken = 0;

	constructor(private props: Props) {
		this.state.steps = props.steps;
		this.state.followUps = props.followUps ?? [];
	}

	/**
	 * Bind Pipeline Composer scroll-follow side effects without a mailbox `$state`.
	 * Pass `{}` on teardown.
	 */
	bindComposerScroll(handlers: {
		onRevealStep?: (index: number) => void;
		onEditFocus?: (stepIndex: number) => void;
	}) {
		this.#composerScroll = handlers;
	}

	#requestReveal(index: number) {
		this.#composerScroll.onRevealStep?.(index);
	}

	#requestEditFocus(stepIndex: number) {
		this.#composerScroll.onEditFocus?.(stepIndex);
	}

	// Shortcuts

	get mode() {
		return this.state.mode;
	}

	get steps() {
		return this.state.steps;
	}

	get followUps() {
		return this.state.followUps;
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
		this.openForm('edit', config, { initial: data, stepIndex: index, section: 'steps' });
		this.#requestEditFocus(index);
	}

	initEditFollowUp(index: number) {
		if (this.state.mode.id === 'form') {
			this.exitFormState();
		}
		const followUp = this.state.followUps[index];
		if (!followUp) return;
		const step = followUp.step;
		if (!isStepEditable(step)) return;
		const config = pipelinestep.getConfigByType(step[0].use);
		const data = getStepData(step);
		if (!config || !data) return;
		this.openForm('edit', config, { initial: data, stepIndex: index, section: 'follow-ups' });
	}

	private openForm(
		intent: pipelinestep.FormIntent,
		config: pipelinestep.AnyConfig,
		opts: { initial?: GenericRecord; stepIndex?: number; section?: FormSection }
	) {
		this.stateManager.run((state) => {
			const effectCleanup = $effect.root(() => {
				let form: pipelinestep.Form;
				try {
					form = config.initForm({
						intent,
						initial: opts.initial as never,
						openStep: (type) => this.initAddStep(type),
						getExecutionTarget: () => this.executionTarget,
						isExecutionTargetLocked: () =>
							isExecutionTargetLocked({
								intent,
								steps: this.state.steps,
								target: this.executionTarget
							}),
						canChangeWalletVersion: () => this.canOfferChangeWalletVersion(),
						requestChangeWalletVersion: () => this.openChangeWalletVersion()
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
								if (!this.placeNewStep(inner, config, formData, 'steps')) return;
								this.#requestReveal(inner.steps.length - 1);
							} else {
								const editIndex = inner.mode.stepIndex;
								if (editIndex === undefined) return;

								const applied =
									inner.mode.section === 'follow-ups'
										? this.applyEditFollowUp(inner, config, formData, editIndex)
										: this.applyEditStep(inner, config, formData, editIndex);
								if (!applied) return;
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
					section: opts.section,
					config,
					form
				};
			});
			this.formEffectCleanup = effectCleanup;
		});
	}

	private createStep(config: pipelinestep.AnyConfig, formData: unknown): PipelineStep {
		return {
			use: config.use as never,
			id: '',
			continue_on_error: false,
			with: config.serialize(formData)
		};
	}

	private isFollowUpUse(use: string) {
		return use === 'email' || use === 'http-request';
	}

	private placeNewStep(
		state: State,
		config: pipelinestep.AnyConfig,
		formData: unknown,
		section: FormSection
	): boolean {
		const step = this.createStep(config, formData);
		if (section === 'follow-ups') {
			if (!this.isFollowUpUse(step.use)) return false;
			state.followUps.push({
				step: [step, formData as GenericRecord],
				condition: 'always'
			});
			this.markCreatedCard('follow-ups', state.followUps.length - 1);
			return true;
		}

		state.steps.push([step, formData as GenericRecord]);
		this.markCreatedCard('steps', state.steps.length - 1);
		return true;
	}

	private applyEditStep(
		state: State,
		config: pipelinestep.AnyConfig,
		formData: unknown,
		index: number
	): boolean {
		const tuple = state.steps[index];
		if (!tuple || tuple[0].use === 'debug') return false;
		tuple[0].with = config.serialize(formData);
		tuple[1] = formData as GenericRecord;
		return true;
	}

	private applyEditFollowUp(
		state: State,
		config: pipelinestep.AnyConfig,
		formData: unknown,
		index: number
	): boolean {
		const followUp = state.followUps[index];
		if (!followUp) return false;
		const [pipelineStep] = followUp.step;
		if (pipelineStep.use === 'debug') return false;
		pipelineStep.with = config.serialize(formData);
		followUp.step[1] = formData as GenericRecord;
		return true;
	}

	isFollowUpEligibleForm() {
		const mode = this.state.mode;
		return mode.id === 'form' && mode.intent === 'add' && this.isFollowUpUse(mode.config.use);
	}

	addAsFollowUp() {
		if (this.state.mode.id !== 'form' || this.state.mode.intent !== 'add') return;
		const { config, form } = this.state.mode;
		if (!this.isFollowUpUse(config.use)) return;

		try {
			const formData = form.getSubmitData();
			if (formData === undefined) return;

			this.stateManager.run((inner) => {
				if (inner.mode.id !== 'form' || inner.mode.intent !== 'add') return;
				if (!this.placeNewStep(inner, config, formData, 'follow-ups')) return;
				inner.mode = { id: 'idle' };
			});
			this.disposeFormEffect();
		} catch (e) {
			showPipelineFormError(e);
		}
	}

	private markCreatedCard(section: CreatedCardRef['section'], index: number) {
		this.createdCard = { section, index, token: ++this.createdCardToken };
	}

	addDebugStep() {
		this.stateManager.run((state) => {
			state.steps.push([{ use: 'debug' }, {}]);
			this.markCreatedCard('steps', state.steps.length - 1);
		});
		this.#requestReveal(this.steps.length - 1);
	}

	deleteStep(index: number) {
		if (this.isFormMode) return;
		this.stateManager.run((state) => {
			state.steps.splice(index, 1);
		});
	}

	deleteFollowUp(index: number) {
		if (this.isFormMode) return;
		this.stateManager.run((state) => {
			state.followUps.splice(index, 1);
		});
	}

	cloneStep(index: number) {
		if (this.isFormMode) return;
		let insertedAt: number | null = null;
		this.stateManager.run((state) => {
			const source = state.steps[index];
			if (!source) return;
			const [pipelineStep, formData] = cloneDeep(source);
			if (pipelineStep.use !== 'debug' && 'id' in pipelineStep) {
				pipelineStep.id = '';
			}
			state.steps.splice(index + 1, 0, [pipelineStep, formData]);
			insertedAt = index + 1;
			this.markCreatedCard('steps', insertedAt);
		});
		if (insertedAt != null) this.#requestReveal(insertedAt);
	}

	setContinueOnError(index: number, continueOnError: boolean) {
		this.stateManager.run((state) => {
			const step = state.steps[index];
			if (!step || step[0].use == 'debug') return;
			step[0].continue_on_error = continueOnError;
		});
	}

	setFollowUpCondition(index: number, condition: EnrichedFollowUp['condition']) {
		this.stateManager.run((state) => {
			const followUp = state.followUps[index];
			if (!followUp) return;
			followUp.condition = condition;
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

	canOfferChangeWalletVersion() {
		const mode = this.state.mode;
		const locked =
			mode.id === 'form' &&
			isExecutionTargetLocked({
				intent: mode.intent,
				steps: this.state.steps,
				target: this.executionTarget
			});
		return isChangeWalletVersionAvailable(this.state.steps, { locked });
	}

	openChangeWalletVersion() {
		this.changeWalletVersionDialogOpen = true;
	}

	applyBulkWalletVersion(version: SelectedVersion) {
		const ctx = getBulkWalletVersionContext(this.state.steps);
		if (!ctx) return;
		this.stateManager.run((state) => {
			state.steps = this.syncMobileStepVersions(state.steps, ctx.wallet.id, version);
			if (state.mode.id === 'form') {
				state.mode.form.applyBulkWalletVersion?.(ctx.wallet.id, version);
			}
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
