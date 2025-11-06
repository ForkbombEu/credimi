// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$lib/marketplace/index.js';

import {
	type CredentialsResponse,
	type CustomChecksResponse,
	type UseCasesVerificationsResponse,
	type WalletActionsResponse,
	type WalletVersionsResponse
} from '@/pocketbase/types/index.generated.js';

/* Steps types */

export type BaseStep<T extends StepType, Data extends Record<string, unknown>> = {
	type: T;
	id: string;
	name: string;
	path: string;
	organization: string;
	recordId: string;
	data: Data;
	continueOnError?: boolean;
};

export enum StepType {
	WalletAction = 'wallet_actions',
	Credential = 'credentials',
	UseCaseVerification = 'use_cases_verifications',
	CustomCheck = 'custom_checks'
}

export type MarketplaceStepType =
	| StepType.Credential
	| StepType.CustomCheck
	| StepType.UseCaseVerification;

//

export type BuilderStep =
	| WalletActionStep
	| CredentialStep
	| CustomCheckStep
	| UseCaseVerificationStep;

export type WalletActionStep = BaseStep<StepType.WalletAction, WalletStepData>;

export type CredentialStep = BaseStep<StepType.Credential, CredentialsResponse>;

export type CustomCheckStep = BaseStep<StepType.CustomCheck, CustomChecksResponse>;

export type UseCaseVerificationStep = BaseStep<
	StepType.UseCaseVerification,
	UseCasesVerificationsResponse
>;

export type WalletStepData = {
	wallet: MarketplaceItem;
	version: WalletVersionsResponse;
	action: WalletActionsResponse;
};

/* Builder states */

export abstract class StepFormState {}

export class IdleState {}
