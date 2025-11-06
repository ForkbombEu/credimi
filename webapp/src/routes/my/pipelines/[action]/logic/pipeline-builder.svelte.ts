// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import type { CollectionResponses } from '@/pocketbase/types/index.generated.js';
import { create } from 'mutative';
import { nanoid } from 'nanoid';
import { BaseStepForm } from './base-step-form.svelte.js';
import type { BuilderStep, MarketplaceStepType, WalletStepData } from './types';
import { IdleState, StepFormState, StepType } from './types';
import { WalletStepForm } from './wallet-step-form.svelte.js';

//

type PipelineBuilderData = {
	steps: BuilderStep[];
	lastWallet: CurrentWallet | undefined;
	state: BuilderState;
};

type History = {
	past: PipelineBuilderData[];
	future: PipelineBuilderData[];
};

export class PipelineBuilder {
	private data = $state<PipelineBuilderData>({
		steps: [],
		lastWallet: undefined,
		state: new IdleState()
	});

	constructor(steps: BuilderStep[] = []) {
		this.data.steps = steps;
	}

	get state() {
		return this.data.state;
	}

	get steps() {
		return this.data.steps;
	}

	readonly yaml = $derived.by(() => {
		return this.steps.map((s) => s.yaml).join('\n---\n');
	});

	// State management

	private history: History = {
		past: [],
		future: []
	};

	private run(action: (data: PipelineBuilderData) => void) {
		this.history.past.push(this.data);
		const nextData = create(this.data, action);
		this.data = nextData;
		this.history.future = [];
	}

	undo() {
		const previousData = this.history.past.pop();
		if (!previousData) return;
		this.history.future.push(this.data);
		this.data = previousData;
	}

	redo() {
		const nextData = this.history.future.pop();
		if (!nextData) return;
		this.history.past.push(this.data);
		this.data = nextData;
	}

	//

	discardAddStep() {
		if (!(this.state instanceof StepFormState)) return;
		this.run((data) => {
			data.state = new IdleState();
		});
	}

	// Needed for Svelte 5 reactivity
	private effectCleanup: (() => void) | undefined = undefined;

	initAddStep(type: StepType) {
		this.effectCleanup = $effect.root(() => {
			if (type === StepType.WalletAction) {
				this.initWalletStepForm();
			} else if (type === StepType.Credential) {
				this.initBaseStepForm(type, async (collection, id) => {
					const data = await pb.collection(collection).getOne(id);
					return {
						data: data,
						yaml: data.yaml ?? data.deeplink
					};
				});
			} else if (type === StepType.CustomCheck) {
				this.initBaseStepForm(type, async (collection, id) => {
					const data = await pb.collection(collection).getOne(id);
					return {
						data: data,
						yaml: data.yaml
					};
				});
			} else if (type === StepType.UseCaseVerification) {
				this.initBaseStepForm(type, async (collection, id) => {
					const data = await pb.collection(collection).getOne(id);
					return {
						data: data,
						yaml: data.yaml
					};
				});
			}
		});
	}

	private addStep(step: Omit<BuilderStep, 'id'>) {
		this.run((data) => {
			data.steps.push({ ...step, id: nanoid(5) } as BuilderStep);
			data.state = new IdleState();
			this.effectCleanup?.();
		});
	}

	private currentWallet: CurrentWallet | undefined = undefined;

	private initWalletStepForm() {
		this.run((data) => {
			data.state = new WalletStepForm({
				initialData: this.currentWallet,
				onSelect: (data: WalletStepData) => {
					this.currentWallet = {
						wallet: data.wallet,
						version: data.version
					};
					this.addStep({
						name: data.action.name,
						path: data.wallet.path + '/' + data.action.canonified_name,
						organization: data.wallet.organization_name,
						data: data,
						type: StepType.WalletAction,
						yaml: data.action.code,
						recordId: data.action.id
					});
				}
			});
		});
	}

	private initBaseStepForm<T extends MarketplaceStepType>(
		collection: T,
		getter: (
			collection: T,
			id: string
		) => Promise<{ data: CollectionResponses[T]; yaml: string }>
	) {
		this.run((data) => {
			data.state = new BaseStepForm({
				collection,
				onSelect: async (item) => {
					const { data, yaml } = await getter(collection, item.id);
					this.addStep({
						name: item.name,
						path: item.path,
						organization: item.organization_name,
						data: data as never,
						type: collection,
						yaml: yaml,
						recordId: item.id
					});
				}
			});
		});
	}

	//

	deleteStep(step: BuilderStep) {
		this.run((data) => {
			data.steps = data.steps.filter((s) => s.id !== step.id);
		});
	}

	shiftStep(item: BuilderStep, change: number) {
		this.run((data) => {
			const indices = this.calculateShiftIndices(item, change, data.steps);
			if (!indices) return;
			const [movedItem] = data.steps.splice(indices.index, 1);
			data.steps.splice(indices.newIndex, 0, movedItem);
		});
	}

	canShiftStep(item: BuilderStep, change: number) {
		return this.calculateShiftIndices(item, change, this.steps) !== null;
	}

	private calculateShiftIndices(item: BuilderStep, change: number, steps: BuilderStep[]) {
		const index = steps.findIndex((s) => s.id === item.id);
		if (index === -1) return null;
		const newIndex = index + change;
		if (newIndex < 0 || newIndex >= steps.length || newIndex === index) return null;
		return { index, newIndex };
	}

	isReady() {
		return this.steps.length > 0;
	}
}

// Types

type BuilderState = IdleState | StepFormState;

type CurrentWallet = Omit<WalletStepData, 'action'>;
