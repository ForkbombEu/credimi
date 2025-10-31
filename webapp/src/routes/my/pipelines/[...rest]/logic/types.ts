// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';

export enum StepType {
	Wallet = 'wallet',
	Credential = 'credential',
	CustomCheck = 'custom_check',
	UseCaseVerification = 'use_case_verification'
}

const WalletStepSchema = z.object({
	type: z.literal(StepType.Wallet),
	walletId: z.string(),
	versionId: z.string(),
	actionId: z.string()
});

const CredentialStepSchema = z.object({
	type: z.literal(StepType.Credential),
	credentialId: z.string()
});

const CustomCheckStepSchema = z.object({
	type: z.literal(StepType.CustomCheck),
	customCheckId: z.string()
});

const UseCaseVerificationStepSchema = z.object({
	type: z.literal(StepType.UseCaseVerification),
	useCaseVerificationId: z.string()
});

export const StepSchema = z
	.discriminatedUnion('type', [
		WalletStepSchema,
		CredentialStepSchema,
		CustomCheckStepSchema,
		UseCaseVerificationStepSchema
	])
	.and(
		z.object({
			id: z.string()
		})
	);

export const StepsSchema = z.array(StepSchema);

export type Step = z.infer<typeof StepSchema>;
