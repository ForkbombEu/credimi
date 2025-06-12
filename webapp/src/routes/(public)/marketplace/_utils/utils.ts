// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { localizeHref, m } from '@/i18n';
import { pb } from '@/pocketbase';
import type { CollectionName } from '@/pocketbase/collections-models';
import MarketplaceItemTypeDisplay from './marketplace-item-type-display.svelte';
import MarketplaceItemCard from './marketplace-item-card.svelte';
import { z } from 'zod';

//

export { MarketplaceItemTypeDisplay, MarketplaceItemCard };

/* -- Marketplace item types -- */

export const marketplaceItemTypes = [
	'verifiers',
	'credential_issuers',
	'wallets',
	'credentials',
	'custom_checks',
	'use_cases_verifications'
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
	bgClass: string;
	textClass: string;
};

type MarketplaceItemsDisplayConfig = {
	[Type in MarketplaceItemType]: MarketplaceItemDisplayData;
};

const marketplaceItemsDisplayConfig: MarketplaceItemsDisplayConfig = {
	wallets: {
		label: m.Wallet(),
		bgClass: 'bg-blue-500',
		textClass: 'text-blue-500'
	},
	verifiers: {
		label: m.Verifier(),
		bgClass: 'bg-green-500',
		textClass: 'text-green-500'
	},
	credential_issuers: {
		label: m.Credential_issuer(),
		bgClass: 'bg-yellow-500',
		textClass: 'text-yellow-500'
	},
	credentials: {
		label: m.Credential(),
		bgClass: 'bg-purple-500',
		textClass: 'text-purple-500'
	},
	custom_checks: {
		label: m.Custom_check(),
		bgClass: 'bg-orange-500',
		textClass: 'text-orange-500'
	},
	use_cases_verifications: {
		label: m.Use_case_verification(),
		bgClass: 'bg-red-500',
		textClass: 'text-red-500'
	}
};

export function getMarketplaceItemTypeData(type: MarketplaceItemType) {
	const display = marketplaceItemsDisplayConfig[type];
	const filter = `type = '${type}'`;
	return { display, filter };
}

export function getMarketplaceItemData(item: MarketplaceItem) {
	const href =
		item.type == 'custom_checks'
			? `/my/tests/new?test_id=${item.id}`
			: localizeHref(`/marketplace/${item.type}/${item.id}`);
	const logo = item.avatar
		? pb.files.getURL(item.avatar, item.avatar.image_file)
		: item.avatar_url
			? item.avatar_url
			: undefined;
	return { href, logo, ...getMarketplaceItemTypeData(item.type) };
}
