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

export { getHubItemByPath, invalidateHubItemByPathCache } from './get-hub-item.js';

//

export function getHubItemTypeEntityData(type: HubItemType): EntityData {
	const display = entities[type];
	if (!display) throw new Error(`Display data not found for item type: ${type}`);
	return display;
}

export function getHubItemTypeFilter(type: HubItemType): string {
	return `type = '${type}'`;
}

/** `thumb`: a PocketBase thumb size for small avatars; 100x100 is the one every file field has. */
export function getHubItemLogo(item: HubItem, thumb?: string): string | undefined {
	if (item.avatar_file) {
		return pb.files.getURL(
			item.type === 'pipelines'
				? { collectionName: 'organizations', id: item.organization_id }
				: { collectionName: item.type, id: item.id },
			item.avatar_file,
			thumb ? { thumb } : undefined
		);
	}
	if (item.avatar_url) {
		if (!thumb) return item.avatar_url;
		// credimi.io serves `?thumb=100x100` on file URLs; append for remote logos too.
		const sep = item.avatar_url.includes('?') ? '&' : '?';
		return `${item.avatar_url}${sep}thumb=${thumb}`;
	}
	return undefined;
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

export function getHubItemData(item: HubItem, thumb?: string) {
	return {
		href: getHubItemUrl(item),
		logo: getHubItemLogo(item, thumb),
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
