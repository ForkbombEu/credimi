// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import type {
	CredentialsResponse,
	CustomChecksResponse,
	UseCasesVerificationsResponse
} from '@/pocketbase/types/index.generated.js';
import { nanoid } from 'nanoid';
import { BaseStepForm } from './base-step-form.svelte.js';
import { IdleState, StepFormState, StepType, type BaseStep } from './types.js';
import { WalletStepForm, type WalletStepData } from './wallet-step-form.svelte.js';

//

export class PipelineBuilder {
	private currentWallet = $state<CurrentWallet>();

	private _steps: BuilderStep[] = $state([]);
	get steps() {
		return this._steps;
	}

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
			} else {
				this.initBaseStepForm(type);
			}
		});
	}

	addStep(step: Omit<BuilderStep, 'id'>) {
		this._steps.push({ ...step, id: nanoid(5) } as BuilderStep);
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
					type: StepType.Wallet
				});
			}
		});
	}

	private initBaseStepForm(collection: StepType) {
		this._state = new BaseStepForm({
			collection,
			onSelect: async (item) => {
				const data = await pb.collection(collection).getOne(item.id);
				this.addStep({
					name: item.name,
					path: item.path,
					organization: item.organization_name,
					data: data as never,
					type: collection
				});
			}
		});
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
