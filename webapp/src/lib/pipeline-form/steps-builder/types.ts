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

export type BuilderStep = WalletActionStep | MarketplaceItemStep;

export type MarketplaceItemStep = BaseStep<MarketplaceStepType, MarketplaceItem>;

export type WalletActionStep = BaseStep<StepType.WalletAction, WalletStepData>;

export type WalletStepData = {
	wallet: MarketplaceItem;
	version: WalletVersionsResponse;
	action: WalletActionsResponse;
};

/* Builder states */

export abstract class StepFormState {}

export class IdleState {}
