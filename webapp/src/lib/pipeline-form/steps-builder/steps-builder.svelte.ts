// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getMarketplaceItemLogo, type MarketplaceItem } from '$lib/marketplace';
import { pb } from '@/pocketbase/index.js';
import { create } from 'mutative';
import { nanoid } from 'nanoid';
import slugify from 'slugify';
import Component from './steps-builder.svelte';
import { BaseStepForm } from './steps/base-step-form.svelte.js';
import { ConformanceCheckStepForm } from './steps/conformance-check-step-form.svelte.js';
import { WalletStepForm } from './steps/wallet-step-form.svelte.js';
import type { BuilderStep, MarketplaceStepType, WalletStepData } from './types';
import { IdleState, StepFormState, StepType } from './types';

//

type StepsBuilderData = {
	steps: BuilderStep[];
	lastWallet: CurrentWallet | undefined;
	state: BuilderState;
};

type Props = {
	steps: BuilderStep[];
	yamlPreview: () => string;
};

export class StepsBuilder {
	readonly Component = Component;

	private data = $state<StepsBuilderData>({
		steps: [],
		lastWallet: undefined,
		state: new IdleState()
	});

	constructor(private props: Props) {
		this.data.steps = props.steps;
	}

	get state() {
		return this.data.state;
	}

	get steps() {
		return this.data.steps;
	}

	get yamlPreview() {
		return this.props.yamlPreview();
	}

	// State management

	private history: History = {
		past: [],
		future: []
	};

	private run(action: (data: StepsBuilderData) => void) {
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
			} else if (type === StepType.ConformanceCheck) {
				this.initConformanceCheckStepForm();
			} else {
				this.initBaseStepForm(type);
			}
		});
	}

	private addStep(step: BuilderStep) {
		this.run((data) => {
			data.steps.push({ ...step, continueOnError: true });
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
					const avatar = getMarketplaceItemLogo(data.wallet);
					this.currentWallet = {
						wallet: data.wallet,
						version: data.version
					};
					this.addStep({
						id: createId(data.action.canonified_name),
						name: data.action.name,
						path: data.wallet.path + '/' + data.action.canonified_name,
						organization: data.wallet.organization_name,
						data: data,
						type: StepType.WalletAction,
						avatar: avatar
					});
				}
			});
		});
	}

	private initBaseStepForm<T extends MarketplaceStepType>(collection: T) {
		this.run((data) => {
			data.state = new BaseStepForm({
				collection,
				onSelect: async (item) => {
					const data: MarketplaceItem = await pb.collection(collection).getOne(item.id);
					const avatar = getMarketplaceItemLogo(data);
					this.addStep({
						id: createId(item.canonified_name),
						name: item.name,
						path: item.path,
						organization: item.organization_name,
						data: data as never,
						type: collection,
						avatar: avatar
					});
				}
			});
		});
	}

	private initConformanceCheckStepForm() {
		this.run((data) => {
			data.state = new ConformanceCheckStepForm({
				onSelect: (checkId: string) => {
					this.addStep({
						id: createId(checkId),
						type: StepType.ConformanceCheck,
						name: checkId,
						path: checkId,
						organization: 'Conformance Check',
						data: { checkId }
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

	setContinueOnError(step: BuilderStep, continueOnError: boolean) {
		this.run((data) => {
			const index = data.steps.findIndex((s) => s.id === step.id);
			if (index === -1) return;
			data.steps[index].continueOnError = continueOnError;
		});
	}

	isReady() {
		return this.steps.length > 0;
	}
}

// Types

type BuilderState = IdleState | StepFormState;

type CurrentWallet = Omit<WalletStepData, 'action'>;

type History = {
	past: StepsBuilderData[];
	future: StepsBuilderData[];
};

// Utils

function createId(base: string): string {
	return slugify(`${base}--${nanoid(5)}`, {
		replacement: '-',
		remove: /[*+~.()'"!:@]/g,
		lower: true,
		strict: true
	});
}
