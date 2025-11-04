// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$lib/marketplace';

import { z } from 'zod';

import { pb } from '@/pocketbase';
import {
	Collections,
	type CredentialsResponse,
	type CustomChecksResponse,
	type UseCasesVerificationsResponse
} from '@/pocketbase/types/index.generated.js';

import type { WalletStepData } from './wallet-step-form.svelte';

/* Builder states */

export abstract class StepFormState {}

export class IdleState {}

/* Pipeline step types */

export type BaseStep<T extends StepType, Data extends Record<string, unknown>> = {
	type: T;
	id: string;
	name: string;
	path: string;
	organization: string;
	yaml: string;
	recordId: string;
	data: Data;
};

export enum StepType {
	Wallet = 'wallets',
	Credential = 'credentials',
	CustomCheck = 'custom_checks',
	UseCaseVerification = 'use_cases_verifications'
}

//

export type BuilderStep = WalletStep | CredentialStep | CustomCheckStep | UseCaseVerificationStep;

type WalletStep = BaseStep<StepType.Wallet, WalletStepData>;
type CredentialStep = BaseStep<StepType.Credential, CredentialsResponse>;
type CustomCheckStep = BaseStep<StepType.CustomCheck, CustomChecksResponse>;
type UseCaseVerificationStep = BaseStep<
	StepType.UseCaseVerification,
	UseCasesVerificationsResponse
>;

/* Serialization */

export const serializedStepSchema = z
	.object({
		type: z.union([
			z.literal(StepType.UseCaseVerification),
			z.literal(StepType.CustomCheck),
			z.literal(StepType.Credential)
		]),
		recordId: z.string()
	})
	.or(
		z.object({
			type: z.literal(StepType.Wallet),
			actionId: z.string(),
			walletId: z.string(),
			versionId: z.string()
		})
	);

export type SerializedStep = z.infer<typeof serializedStepSchema>;

export function serializeStep(step: BuilderStep): SerializedStep {
	if (step.type !== StepType.Wallet) {
		return {
			type: step.type,
			recordId: step.recordId
		};
	} else {
		return {
			type: step.type,
			walletId: step.data.wallet.id,
			versionId: step.data.version.id,
			actionId: step.data.action.id
		};
	}
}

export async function deserializeStep(step: unknown): Promise<BuilderStep> {
	const parsed = serializedStepSchema.parse(step);
	if (parsed.type === StepType.Wallet) {
		const action = await pb.collection('wallet_actions').getOne(parsed.actionId);
		const walletItem: MarketplaceItem = await pb
			.collection('marketplace_items')
			.getFirstListItem(`type = "${Collections.Wallets}" && id = "${action.wallet}"`);
		const version = await pb.collection('wallet_versions').getOne(parsed.versionId);
		return {
			type: StepType.Wallet,
			id: action.id,
			name: action.name,
			path: walletItem.path + '/' + action.canonified_name,
			organization: walletItem.organization_name,
			yaml: action.code,
			recordId: action.id,
			data: {
				wallet: walletItem,
				version: version,
				action: action
			}
		};
	} else if (parsed.type === StepType.Credential) {
		return {
			type: StepType.Credential,
			recordId: parsed.recordId
		};
	} else if (parsed.type === StepType.CustomCheck) {
		return {
			type: StepType.CustomCheck,
			recordId: parsed.recordId
		};
	} else if (parsed.type === StepType.UseCaseVerification) {
		return {
			type: StepType.UseCaseVerification,
			recordId: parsed.recordId
		};
	} else {
		throw new Error('Invalid step type');
	}
}

//

//
