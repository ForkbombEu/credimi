// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { queryParams } from '$routes/my/tests/new/+page.svelte';
import { z } from 'zod';

import type { CollectionName } from '@/pocketbase/collections-models';

import { localizeHref, m } from '@/i18n';
import { pb } from '@/pocketbase';

import MarketplaceItemCard from './marketplace-item-card.svelte';
import MarketplaceItemTypeDisplay from './marketplace-item-type-display.svelte';

//

export { MarketplaceItemTypeDisplay, MarketplaceItemCard };

/* -- Marketplace item types -- */

export const marketplaceItemTypes = [
	'wallets',
	'credential_issuers',
	'credentials',
	'verifiers',
	'use_cases_verifications',
	'custom_checks'
] as const satisfies CollectionName[];

export const marketplaceItemTypeSchema = z.enum(marketplaceItemTypes);
export type MarketplaceItemType = z.infer<typeof marketplaceItemTypeSchema>;

/* -- Marketplace item type -- */

// This type is needed as the MarketplaceItem type coming from codegen is not good.
// Since `marketplace_items` is a view collection, that merges multiple collections,
// pocketbase says that each field is of type `json` and not the actual type.

export type MarketplaceItem = {
	collectionId: string;
	collectionName: string;
	id: string;
	type: MarketplaceItemType;
	name: string;
	description: string | null;
	avatar: { [key: string]: unknown; image_file: string } | null;
	avatar_url: string | null;
	updated: string;
	organization_id: string;
	organization_name: string;
};

/* -- Marketplace item type mapping to display data -- */

export type MarketplaceItemDisplayData = {
	label: string;
	labelPlural: string;
	bgClass: string;
	textClass: string;
};

type MarketplaceItemsDisplayConfig = {
	[Type in MarketplaceItemType]: MarketplaceItemDisplayData;
};

const marketplaceItemsDisplayConfig: MarketplaceItemsDisplayConfig = {
	wallets: {
		label: m.Wallet(),
		labelPlural: m.Wallets(),
		bgClass: 'bg-blue-500',
		textClass: 'text-blue-500'
	},
	custom_checks: {
		label: m.Custom_check(),
		labelPlural: m.Custom_checks(),
		bgClass: 'bg-purple-500',
		textClass: 'text-purple-500'
	},
	credential_issuers: {
		label: m.Credential_issuer(),
		labelPlural: m.Credential_issuers(),
		bgClass: 'bg-green-700',
		textClass: 'text-green-700'
	},
	credentials: {
		label: m.Credential(),
		labelPlural: m.Credentials(),
		bgClass: 'bg-green-400',
		textClass: 'text-green-400'
	},
	verifiers: {
		label: m.Verifier(),
		labelPlural: m.Verifiers(),
		bgClass: 'bg-red-600',
		textClass: 'text-red-600'
	},
	use_cases_verifications: {
		label: m.Use_case_verification(),
		labelPlural: m.Use_case_verifications(),
		bgClass: 'bg-orange-400',
		textClass: 'text-orange-400'
	}
};

export function getMarketplaceItemTypeData(type: MarketplaceItemType) {
	const display = marketplaceItemsDisplayConfig[type];
	const filter = `type = '${type}'`;
	return { display, filter };
}

export function getMarketplaceItemData(item: MarketplaceItem) {
	const href =
		item.type === 'custom_checks'
			? `/my/tests/new?${queryParams.customCheckId}=${item.id}`
			: localizeHref(`/marketplace/${item.type}/${item.id}`);

	const logo = item.avatar
		? pb.files.getURL(item.avatar, item.avatar.image_file)
		: item.avatar_url
			? item.avatar_url
			: undefined;

	return { href, logo, ...getMarketplaceItemTypeData(item.type) };
}
