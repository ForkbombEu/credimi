// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { marketplaceItemsDisplayConfig, type MarketplaceItemDisplayData } from '$lib/marketplace';
import { appSections } from '$lib/marketplace/sections';
import { Mail, Bug, Globe } from 'lucide-svelte';

import { m } from '@/i18n/index.js';

import { StepType } from '../types';

//

const { wallets, credential_issuers, custom_checks, use_cases_verifications } =
	marketplaceItemsDisplayConfig;

const { conformance_checks } = appSections;

const stepDisplayDataMap: Record<StepType, MarketplaceItemDisplayData> = {
	[StepType.WalletAction]: { ...wallets, label: m.Wallet_action() },
	[StepType.Credential]: {
		...credential_issuers,
		label: m.Credential_deeplink()
	},
	[StepType.CustomCheck]: custom_checks,
	[StepType.UseCaseVerification]: use_cases_verifications,
	[StepType.ConformanceCheck]: {
		icon: conformance_checks.icon,
		label: m.Conformance_check(),
		labelPlural: conformance_checks.label,
		bgClass: 'bg-red-500',
		textClass: conformance_checks.textClass,
		backgroundClass: 'bg-red-500',
		outlineClass: 'border-red-500'
	},
	[StepType.Email]: {
		icon: Mail,
		label: m.Utils_Email(),
		labelPlural: m.Utils_Email(),
		bgClass: 'bg-[hsl(var(--cyan-foreground))]',
		textClass: 'text-[hsl(var(--cyan-foreground))]',
		backgroundClass: 'bg-[hsl(var(--cyan-background))]',
		outlineClass: 'border-[hsl(var(--cyan-outline))]'
	},
	[StepType.Debug]: {
		icon: Bug,
		label: m.Utils_Debug(),
		labelPlural: m.Utils_Debug(),
		bgClass: 'bg-[hsl(var(--yellow-foreground))]',
		textClass: 'text-[hsl(var(--yellow-foreground))]',
		backgroundClass: 'bg-[hsl(var(--yellow-background))]',
		outlineClass: 'border-[hsl(var(--yellow-outline))]'
	},
	[StepType.HttpRequest]: {
		icon: Globe,
		label: m.Utils_HTTP_Request(),
		labelPlural: m.Utils_HTTP_Request(),
		bgClass: 'bg-[hsl(var(--indigo-foreground))]',
		textClass: 'text-[hsl(var(--indigo-foreground))]',
		backgroundClass: 'bg-[hsl(var(--indigo-background))]',
		outlineClass: 'border-[hsl(var(--indigo-outline))]'
	}
};

export function getStepDisplayData(stepType: StepType) {
	return stepDisplayDataMap[stepType];
}
