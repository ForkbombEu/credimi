// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export enum StepType {
	Wallet = 'wallets',
	Credential = 'credentials',
	CustomCheck = 'custom_checks',
	UseCaseVerification = 'use_cases_verifications'
}

export type BaseStep<T extends StepType, Data extends Record<string, unknown>> = {
	type: T;
	id: string;
	name: string;
	path: string;
	organization: string;
	data: Data;
};

export abstract class StepFormState {}

export class IdleState {}
