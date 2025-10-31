// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';
import type { Step } from './types';
import { StepType } from './types';

//

export class PipelineBuilder {
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
		this._state = getStepState(type);
	}

	discardAddStep() {
		if (this._state instanceof AddStepState) {
			this._state = new IdleState();
		}
	}
}

type BuilderState = IdleState | AddStepState;

export class IdleState {}

export abstract class AddStepState {
	abstract schema: z.ZodTypeAny;
}

export class AddWalletState extends AddStepState {
	schema = z.object({
		type: z.literal(StepType.Wallet),
		walletId: z.string(),
		versionId: z.string(),
		actionId: z.string()
	});
}

export class AddCredentialState extends AddStepState {
	schema = z.object({
		credentialId: z.string()
	});
}

export class AddCustomCheckState extends AddStepState {
	schema = z.object({
		customCheckId: z.string()
	});
}

export class AddUseCaseVerificationState extends AddStepState {
	schema = z.object({
		useCaseVerificationId: z.string()
	});
}

function getStepState(type: StepType): AddStepState {
	switch (type) {
		case StepType.Wallet:
			return new AddWalletState();
		case StepType.Credential:
			return new AddCredentialState();
		case StepType.CustomCheck:
			return new AddCustomCheckState();
		case StepType.UseCaseVerification:
			return new AddUseCaseVerificationState();
	}
}
