// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities, type EntityData } from '$lib/global/entities';
import {
	credentialIssuersAndCredentialsSection,
	verifiersAndUseCaseVerificationsSection
} from '$lib/global/sections';
import { getPath } from '$lib/utils';

import type { CustomChecksResponse, HubItemsResponse } from '@/pocketbase/types';

import { localizeHref } from '@/i18n';
import { pb } from '@/pocketbase';

import { hubItemTypes, type HubItem, type HubItemType } from './types';

//

export function getHubItemTypeEntityData(type: HubItemType): EntityData {
	const display = entities[type];
	if (!display) throw new Error(`Display data not found for item type: ${type}`);
	return display;
}

export function getHubItemTypeFilter(type: HubItemType): string {
	return `type = '${type}'`;
}

export function getHubItemLogo(item: HubItem): string | undefined {
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

//

export function getCustomCheckPublicUrl(item: HubItem | CustomChecksResponse) {
	return `/my/custom-integrations/${getPath(item, true)}/run`;
}

export function getHubItemUrl(item: HubItem) {
	const href =
		item.type === 'custom_checks' ? getCustomCheckPublicUrl(item) : `/hub/${item.path}`;
	return localizeHref(href);
}

//

export function getHubItemData(item: HubItem) {
	return {
		href: getHubItemUrl(item),
		logo: getHubItemLogo(item),
		display: getHubItemTypeEntityData(item.type)
	};
}

//

export function isCredentialIssuer(item: HubItemsResponse): boolean {
	return item.type === hubItemTypes[1];
}

export function isVerifier(item: HubItemsResponse): boolean {
	return item.type === hubItemTypes[3];
}

//

export function getHubItemByPath(path: string): Promise<HubItem> {
	return pb.collection('hub_items').getFirstListItem(pb.filter('path ~ {:path}', { path }));
}

//

const hubItemTypeToSectionId: Record<HubItemType, string> = {
	wallets: entities.wallets.slug,
	credential_issuers: credentialIssuersAndCredentialsSection.slug,
	credentials: credentialIssuersAndCredentialsSection.slug,
	verifiers: verifiersAndUseCaseVerificationsSection.slug,
	use_cases_verifications: verifiersAndUseCaseVerificationsSection.slug,
	custom_checks: entities.custom_checks.slug,
	pipelines: entities.pipelines.slug
};

export function hubItemToSectionHref(item: HubItem): string {
	const sectionId = hubItemTypeToSectionId[item.type];
	if (!sectionId) throw new Error(`Section not found for item type: ${item.type}`);
	return `/my/${sectionId}`;
}
