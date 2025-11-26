// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$lib/marketplace/index.js';

import {
	type WalletActionsResponse,
	type WalletVersionsResponse
} from '@/pocketbase/types/index.generated.js';

/* Steps types */

export type BaseStep<T extends StepType, Data> = {
	type: T;
	id: string;
	name: string;
	path: string;
	organization: string;
	avatar?: string;
	data: Data;
	continueOnError?: boolean;
	video?: boolean;
};

//

export enum StepType {
	WalletAction = 'wallet_actions',
	Credential = 'credentials',
	UseCaseVerification = 'use_cases_verifications',
	ConformanceCheck = 'conformance_checks',
	CustomCheck = 'custom_checks'
}

export type BuilderStep = WalletActionStep | MarketplaceItemStep | ConformanceCheckStep;

//

export type MarketplaceStepType =
	| StepType.Credential
	| StepType.CustomCheck
	| StepType.UseCaseVerification;

export type MarketplaceItemStep = BaseStep<MarketplaceStepType, MarketplaceItem>;

//

export type WalletActionStep = BaseStep<StepType.WalletAction, WalletStepData>;

export type WalletStepData = {
	wallet: MarketplaceItem;
	version: WalletVersionsResponse;
	action: WalletActionsResponse;
};

//

export type ConformanceCheckStep = BaseStep<StepType.ConformanceCheck, { checkId: string }>;

/* Builder states */

export abstract class StepFormState {}

export class IdleState {}
