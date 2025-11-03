// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export enum StepType {
	Wallet = 'wallets',
	Credential = 'credentials',
	CustomCheck = 'custom_checks',
	UseCaseVerification = 'use_cases_verifications'
}

export type BaseStep<T extends StepType, Data extends Record<string, unknown>> = Data & { type: T };

export abstract class StepFormState {}

export class IdleState {}
