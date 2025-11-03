// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { WalletStepForm, type WalletStepData } from './wallet.svelte.js';

//

export class PipelineBuilder {
	private currentWallet = $state<CurrentWallet>();

	private _steps: Step[] = $state([]);
	get steps() {
		return this._steps;
	}

	private _state: BuilderState = $state(new IdleState());
	get state() {
		return this._state;
	}

	constructor(steps: Step[] = []) {
		this._steps = steps;
	}

	initAddStep(type: StepType) {
		const cleanup = $effect.root(() => {
			if (type === StepType.Wallet) {
				this._state = new WalletStepForm({
					initialData: this.currentWallet,
					onSelect: (data: WalletStepData) => {
						this._steps.push({ ...data, type: StepType.Wallet });
						this.currentWallet = {
							wallet: data.wallet,
							version: data.version
						};
						this._state = new IdleState();
						cleanup();
					}
				});
			} else {
				throw new Error(`Unknown step type: ${type}`);
			}
		});
	}

	discardAddStep() {
		if (this._state instanceof AddStepState) {
			this._state = new IdleState();
		}
	}
}

export enum StepType {
	Wallet = 'wallet',
	Credential = 'credential',
	CustomCheck = 'custom_check',
	UseCaseVerification = 'use_case_verification'
}

type BuilderState = IdleState | AddStepState;

export class IdleState {}

export abstract class AddStepState {}

type BaseStep<T extends StepType, Data extends Record<string, unknown>> = Data & { type: T };
type WalletStep = BaseStep<StepType.Wallet, WalletStepData>;
type Step = WalletStep;

//

type CurrentWallet = Omit<WalletStepData, 'action'>;
