// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities, type EntityData } from '$lib/global';

import { m } from '@/i18n/index.js';

import { StepType } from '../types';

//

const stepDisplayDataMap: Record<StepType, EntityData> = {
	[StepType.WalletAction]: {
		...entities.wallets,
		labels: { singular: m.Wallet_action(), plural: m.Wallet_actions() }
	},
	[StepType.Credential]: {
		...entities.credentials,
		labels: { singular: m.Credential_deeplink(), plural: m.Credential_deeplinks() }
	},
	[StepType.CustomCheck]: entities.custom_checks,
	[StepType.UseCaseVerification]: entities.use_cases_verifications,
	[StepType.ConformanceCheck]: entities.conformance_checks
};

export function getStepDisplayData(stepType: StepType) {
	return stepDisplayDataMap[stepType];
}
