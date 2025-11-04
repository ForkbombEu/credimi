// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import type {
	CollectionResponses,
	CredentialsResponse,
	CustomChecksResponse,
	UseCasesVerificationsResponse
} from '@/pocketbase/types/index.generated.js';
import { create } from 'mutative';
import { nanoid } from 'nanoid';
import { BaseStepForm } from './base-step-form.svelte.js';
import { IdleState, StepFormState, StepType, type BaseStep } from './types.js';
import { WalletStepForm, type WalletStepData } from './wallet-step-form.svelte.js';

//

export class PipelineBuilder {
	private currentWallet = $state<CurrentWallet>();
	public readonly yaml = $derived.by<string>(() => {
		return this.steps.map((step) => step.yaml).join('\n---\n');
	});

	private _steps: BuilderStep[] = $state([]);
	get steps() {
		return this._steps;
	}

	private stepsHistory = {
		past: [] as Array<BuilderStep[]>,
		future: [] as Array<BuilderStep[]>
	};

	private _state: BuilderState = $state(new IdleState());
	get state() {
		return this._state;
	}

	constructor(steps: BuilderStep[] = []) {
		this._steps = steps;
	}

	discardAddStep() {
		if (this._state instanceof StepFormState) {
			this._state = new IdleState();
		}
	}

	// Needed for Svelte 5 reactivity
	private effectCleanup: (() => void) | undefined = undefined;

	initAddStep(type: StepType) {
		this.effectCleanup = $effect.root(() => {
			if (type === StepType.Wallet) {
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

	addStep(step: Omit<BuilderStep, 'id'>) {
		this.run((steps) => {
			steps.push({ ...step, id: nanoid(5) } as BuilderStep);
		});
		this.effectCleanup?.();
		this.effectCleanup = undefined;
		this._state = new IdleState();
	}

	private initWalletStepForm() {
		this._state = new WalletStepForm({
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
					type: StepType.Wallet,
					yaml: data.action.code
				});
			}
		});
	}

	private initBaseStepForm<T extends StepType>(
		collection: T,
		getter: (
			collection: T,
			id: string
		) => Promise<{ data: CollectionResponses[T]; yaml: string }>
	) {
		this._state = new BaseStepForm({
			collection,
			onSelect: async (item) => {
				const { data, yaml } = await getter(collection, item.id);
				this.addStep({
					name: item.name,
					path: item.path,
					organization: item.organization_name,
					data: data as never,
					type: collection,
					yaml: yaml
				});
			}
		});
	}

	//

	deleteStep(step: BuilderStep) {
		this.run((steps) => {
			steps = steps.filter((s) => s.id !== step.id);
		});
	}

	shiftStep(item: BuilderStep, change: number) {
		this.run((steps) => {
			const index = this._steps.indexOf(item);
			if (!this.canShiftStep(item, change)) return;
			steps.splice(index, 1);
			steps.splice(index + change, 0, item);
		});
	}

	canShiftStep(item: BuilderStep, change: number) {
		const index = this._steps.indexOf(item);
		return index !== -1 && (change < 0 ? index > 0 : index < this._steps.length - 1);
	}

	//

	run(action: (steps: BuilderStep[]) => void) {
		const newState = create(this._steps, (state) => {
			action(state);
		});
		this.stepsHistory.past.push(this.steps);
		this.stepsHistory.future = [];
		this._steps = newState;
	}

	undo() {
		this._steps = this.stepsHistory.past.pop() || [];
		this.stepsHistory.future.push(this._steps);
		console.log(this.stepsHistory);
	}

	redo() {
		this._steps = this.stepsHistory.future.pop() || [];
		this.stepsHistory.past.push(this._steps);
	}
}

// Types

type BuilderState = IdleState | StepFormState;

export type BuilderStep = WalletStep | CredentialStep | CustomCheckStep | UseCaseVerificationStep;

type WalletStep = BaseStep<StepType.Wallet, WalletStepData>;
type CredentialStep = BaseStep<StepType.Credential, CredentialsResponse>;
type CustomCheckStep = BaseStep<StepType.CustomCheck, CustomChecksResponse>;
type UseCaseVerificationStep = BaseStep<
	StepType.UseCaseVerification,
	UseCasesVerificationsResponse
>;

type CurrentWallet = Omit<WalletStepData, 'action'>;
