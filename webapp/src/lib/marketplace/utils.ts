// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities, type EntityData } from '$lib/global/entities';
import {
	credentialIssuersAndCredentialsSection,
	verifiersAndUseCaseVerificationsSection
} from '$lib/global/sections';

import type { MarketplaceItemsResponse } from '@/pocketbase/types';

import { localizeHref } from '@/i18n';
import { pb } from '@/pocketbase';

import { marketplaceItemTypes, type MarketplaceItem, type MarketplaceItemType } from './types';

//

export function getMarketplaceItemTypeEntityData(type: MarketplaceItemType): EntityData {
	const display = entities[type];
	if (!display) throw new Error(`Display data not found for item type: ${type}`);
	return display;
}

export function getMarketplaceItemTypeFilter(type: MarketplaceItemType): string {
	return `type = '${type}'`;
}

export function getMarketplaceItemLogo(item: MarketplaceItem): string | undefined {
	return item.avatar_file
		? pb.files.getURL(
				item.type === 'pipelines'
					? { collectionName: 'organizations', id: item.organization_id }
					: { collectionName: item.type, id: item.id },
				item.avatar_file
			)
		: item.avatar_url
			? item.avatar_url
			: undefined;
}

export const CUSTOM_CHECK_QUERY_PARAM = 'custom_check_id';

export function getMarketplaceItemUrl(item: MarketplaceItem) {
	const href =
		item.type === 'custom_checks'
			? `/my/tests/new?${CUSTOM_CHECK_QUERY_PARAM}=${item.id}`
			: `/marketplace/${item.path}`;
	return localizeHref(href);
}

//

export function getMarketplaceItemData(item: MarketplaceItem) {
	return {
		href: getMarketplaceItemUrl(item),
		logo: getMarketplaceItemLogo(item),
		display: getMarketplaceItemTypeEntityData(item.type)
	};
}

//

export function isCredentialIssuer(item: MarketplaceItemsResponse): boolean {
	return item.type === marketplaceItemTypes[1];
}

export function isVerifier(item: MarketplaceItemsResponse): boolean {
	return item.type === marketplaceItemTypes[3];
}

//

export function getMarketplaceItemByPath(path: string): Promise<MarketplaceItem> {
	return pb
		.collection('marketplace_items')
		.getFirstListItem(pb.filter('path ~ {:path}', { path }));
}

//

const marketplaceItemTypeToSectionId: Record<MarketplaceItemType, string> = {
	wallets: entities.wallets.slug,
	credential_issuers: credentialIssuersAndCredentialsSection.slug,
	credentials: credentialIssuersAndCredentialsSection.slug,
	verifiers: verifiersAndUseCaseVerificationsSection.slug,
	use_cases_verifications: verifiersAndUseCaseVerificationsSection.slug,
	custom_checks: entities.custom_checks.slug,
	pipelines: entities.pipelines.slug
};

export function marketplaceItemToSectionHref(item: MarketplaceItem): string {
	const sectionId = marketplaceItemTypeToSectionId[item.type];
	if (!sectionId) throw new Error(`Section not found for item type: ${item.type}`);
	return `/my/${sectionId}`;
}
