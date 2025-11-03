// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';
import type {
	CredentialsResponse,
	CustomChecksResponse,
	UseCasesVerificationsResponse
} from '@/pocketbase/types/index.generated.js';
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
				this._state = new WalletStepForm({
					initialData: this.currentWallet,
					onSelect: (data: WalletStepData) => {
						this.currentWallet = {
							wallet: data.wallet,
							version: data.version
						};
						this.finalizeAddStep({ ...data, type: StepType.Wallet });
					}
				});
			} else if (type === StepType.Credential) {
				this._state = new BaseStepForm({
					collection: 'credentials',
					onSelect: async (item) => {
						const credential = await pb.collection('credentials').getOne(item.id);
						this.finalizeAddStep({ credential, type: StepType.Credential });
					}
				});
			} else if (type === StepType.CustomCheck) {
				this._state = new BaseStepForm({
					collection: 'custom_checks',
					onSelect: async (item) => {
						const customCheck = await pb.collection('custom_checks').getOne(item.id);
						this.finalizeAddStep({ customCheck, type: StepType.CustomCheck });
					}
				});
			} else if (type === StepType.UseCaseVerification) {
				this._state = new BaseStepForm({
					collection: 'use_cases_verifications',
					onSelect: async (item) => {
						const useCaseVerification = await pb
							.collection('use_cases_verifications')
							.getOne(item.id);
						this.finalizeAddStep({
							useCaseVerification,
							type: StepType.UseCaseVerification
						});
					}
				});
			} else {
				throw new Error(`Unknown step type: ${type}`);
			}
		});
	}

	finalizeAddStep(step: BuilderStep) {
		this._steps.push(step);
		this.effectCleanup?.();
		this.effectCleanup = undefined;
		this._state = new IdleState();
	}
}

// Types

type BuilderState = IdleState | StepFormState;

type BuilderStep = WalletStep | CredentialStep | CustomCheckStep | UseCaseVerificationStep;

type WalletStep = BaseStep<StepType.Wallet, WalletStepData>;
type CredentialStep = BaseStep<StepType.Credential, { credential: CredentialsResponse }>;
type CustomCheckStep = BaseStep<StepType.CustomCheck, { customCheck: CustomChecksResponse }>;
type UseCaseVerificationStep = BaseStep<
	StepType.UseCaseVerification,
	{ useCaseVerification: UseCasesVerificationsResponse }
>;

type CurrentWallet = Omit<WalletStepData, 'action'>;
