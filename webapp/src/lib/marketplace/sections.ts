// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { IconComponent } from '@/components/types';

import { m } from '@/i18n';

import {
	marketplaceItemsDisplayConfig,
	type MarketplaceItem,
	type MarketplaceItemType
} from './utils';

//

type AppSection = {
	label: string;
	id: string;
	icon: IconComponent;
	textClass: string;
};

export const appSections = {
	wallets: {
		label: m.Wallets(),
		id: 'wallets',
		icon: marketplaceItemsDisplayConfig.wallets.icon,
		textClass: marketplaceItemsDisplayConfig.wallets.textClass
	},
	credential_issuers: {
		label: `${m.Credential_issuers()} / ${m.Credentials()}`,
		id: 'credential-issuers-and-credentials',
		icon: marketplaceItemsDisplayConfig.credential_issuers.icon,
		textClass: marketplaceItemsDisplayConfig.credential_issuers.textClass
	},
	verifiers: {
		label: ` ${m.Verifiers()} / ${m.Use_case_verifications()}`,
		id: 'verifiers-and-use-case-verifications',
		icon: marketplaceItemsDisplayConfig.verifiers.icon,
		textClass: marketplaceItemsDisplayConfig.verifiers.textClass
	},
	custom_checks: {
		label: m.Custom_checks(),
		id: 'custom-checks',
		icon: marketplaceItemsDisplayConfig.custom_checks.icon,
		textClass: marketplaceItemsDisplayConfig.custom_checks.textClass
	}
} as const satisfies Record<string, AppSection>;

//

type SectionId = (typeof appSections)[keyof typeof appSections]['id'];

const marketplaceTypeToSectionId: Record<MarketplaceItemType, SectionId> = {
	wallets: 'wallets',
	credential_issuers: 'credential-issuers-and-credentials',
	verifiers: 'verifiers-and-use-case-verifications',
	custom_checks: 'custom-checks',
	use_cases_verifications: 'verifiers-and-use-case-verifications',
	credentials: 'credential-issuers-and-credentials'
};

export function marketplaceItemToSectionHref(item: MarketplaceItem): string {
	const sectionId = marketplaceTypeToSectionId[item.type];
	return `/my/${sectionId}`;
}
