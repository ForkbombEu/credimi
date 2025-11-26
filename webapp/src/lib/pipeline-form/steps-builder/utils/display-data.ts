// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { marketplaceItemsDisplayConfig, type MarketplaceItemDisplayData } from '$lib/marketplace';
import { SheetIcon } from 'lucide-svelte';

import { m } from '@/i18n/index.js';

import { StepType } from '../types';

//

const { wallets, credential_issuers, custom_checks, use_cases_verifications } =
	marketplaceItemsDisplayConfig;

const stepDisplayDataMap: Record<StepType, MarketplaceItemDisplayData> = {
	[StepType.WalletAction]: { ...wallets, label: m.Wallet_action() },
	[StepType.Credential]: {
		...credential_issuers,
		label: m.Credential_deeplink()
	},
	[StepType.CustomCheck]: custom_checks,
	[StepType.UseCaseVerification]: use_cases_verifications,
	[StepType.ConformanceCheck]: {
		icon: SheetIcon,
		label: m.Conformance_check(),
		labelPlural: m.Conformance_Checks(),
		bgClass: 'bg-red-500',
		textClass: 'text-red-500',
		backgroundClass: 'bg-red-500',
		outlineClass: 'border-red-500'
	}
};

export function getStepDisplayData(stepType: StepType) {
	return stepDisplayDataMap[stepType];
}
