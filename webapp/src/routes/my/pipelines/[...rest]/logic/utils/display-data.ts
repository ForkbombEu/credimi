// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { marketplaceItemsDisplayConfig, type MarketplaceItemDisplayData } from '$lib/marketplace';

import { m } from '@/i18n/index.js';

import { StepType } from '../pipeline-builder.svelte.js';

//

const { wallets, credential_issuers, custom_checks, use_cases_verifications } =
	marketplaceItemsDisplayConfig;

const stepDisplayDataMap: Record<StepType, MarketplaceItemDisplayData> = {
	[StepType.Wallet]: { ...wallets, label: m.Wallet_action() },
	[StepType.Credential]: {
		...credential_issuers,
		label: m.Credential_deeplink()
	},
	[StepType.CustomCheck]: custom_checks,
	[StepType.UseCaseVerification]: use_cases_verifications
};

export function getStepDisplayData(stepType: StepType) {
	return stepDisplayDataMap[stepType];
}
