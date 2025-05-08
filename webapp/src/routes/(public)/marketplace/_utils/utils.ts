// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { localizeHref, m } from '@/i18n';
import { pb } from '@/pocketbase';
import type { CollectionName } from '@/pocketbase/collections-models';
import MarketplaceItemTypeDisplay from './marketplace-item-type-display.svelte';

export { MarketplaceItemTypeDisplay };

/**
 * This type is needed as the MarketplaceItem type coming from codegen is not good.
 * Since `marketplace_items` is a view collection, that merges multiple collections,
 * pocketbase says that each field is of type `json` and not the actual type.
 */
export type MarketplaceItem = {
	collectionId: string;
	collectionName: string;
	id: string;
	type: CollectionName;
	name: string;
	description: string | null;
	avatar: string | null;
	avatar_url: string | null;
	updated: string;
};

export type MarketplaceItemDisplayData = {
	label: string;
	bgClass: string;
	textClass: string;
};

const displayData: Partial<Record<CollectionName, MarketplaceItemDisplayData>> = {
	wallets: { label: m.Wallet(), bgClass: 'bg-blue-500', textClass: 'text-blue-500' },
	verifiers: { label: m.Verifier(), bgClass: 'bg-green-500', textClass: 'text-green-500' },
	credential_issuers: {
		label: m.Credential_issuer(),
		bgClass: 'bg-yellow-500',
		textClass: 'text-yellow-500'
	}
};

export function getMarketplaceItemTypeData(type: CollectionName) {
	const display = displayData[type];
	const filter = `type = '${type}'`;
	return { display, filter };
}

export function getMarketplaceItemData(item: MarketplaceItem) {
	const href = localizeHref(`/marketplace/${item.type}/${item.id}`);
	const logo = item.avatar ? pb.files.getURL(item, item.avatar) : item.avatar_url;
	return { href, logo, ...getMarketplaceItemTypeData(item.type) };
}
