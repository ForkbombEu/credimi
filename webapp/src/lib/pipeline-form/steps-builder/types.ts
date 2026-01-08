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
	CustomCheck = 'custom_checks',
	Email = 'email',
	Debug = 'debug',
	HttpRequest = 'http_request'
}

export type BuilderStep =
	| WalletActionStep
	| MarketplaceItemStep
	| ConformanceCheckStep
	| UtilityStep;

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

//

export type UtilityStepType = StepType.Email | StepType.Debug | StepType.HttpRequest;

export type EmailStepData = {
	recipient: string;
	subject?: string;
	body?: string;
};

export type DebugStepData = Record<string, never>; // Empty object, debug doesn't need specific data

export type HttpRequestStepData = {
	method: string;
	url: string;
	headers?: Record<string, string>;
	body?: string;
};

export type EmailStep = BaseStep<StepType.Email, EmailStepData>;
export type DebugStep = BaseStep<StepType.Debug, DebugStepData>;
export type HttpRequestStep = BaseStep<StepType.HttpRequest, HttpRequestStepData>;

export type UtilityStep = EmailStep | DebugStep | HttpRequestStep;

/* Builder states */

export abstract class StepFormState {}

export class IdleState {}
