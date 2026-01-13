// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$lib/marketplace/types.js';

import { entities } from '$lib/global/entities.js';
import { getMarketplaceItemByPath, getMarketplaceItemLogo } from '$lib/marketplace/utils.js';
import { getPath } from '$lib/utils';

import { m } from '@/i18n/index.js';

import type { TypedConfig } from '../types';

import { getLastPathSegment } from '../_partials/misc';
import { MarketplaceItemStepForm } from './marketplace-item-step-form.svelte.js';

//

export const credentialsStepConfig: TypedConfig<'credential-offer', MarketplaceItem> = {
	use: 'credential-offer',
	display: {
		...entities.credentials,
		labels: { ...entities.credentials.labels, singular: m.Credential_Deeplink() }
	},
	initForm: () =>
		new MarketplaceItemStepForm({
			collection: 'credentials',
			entityData: entities.credentials
		}),
	serialize: (item) => ({ credential_id: getPath(item) }),
	deserialize: ({ credential_id }) => getMarketplaceItemByPath(credential_id),
	cardData: getMarketplaceItemCardData,
	makeId: ({ credential_id }) => getLastPathSegment(credential_id)
};

//

export const useCaseVerificationStepConfig: TypedConfig<
	'use-case-verification-deeplink',
	MarketplaceItem
> = {
	use: 'use-case-verification-deeplink',
	display: entities.use_cases_verifications,
	initForm: () =>
		new MarketplaceItemStepForm({
			collection: 'use_cases_verifications',
			entityData: entities.use_cases_verifications
		}),
	serialize: (item) => ({ use_case_id: getPath(item) }),
	deserialize: ({ use_case_id }) => getMarketplaceItemByPath(use_case_id),
	cardData: getMarketplaceItemCardData,
	makeId: ({ use_case_id }) => getLastPathSegment(use_case_id)
};

//

export const customCheckStepConfig: TypedConfig<'custom-check', MarketplaceItem> = {
	use: 'custom-check',
	display: entities.custom_checks,
	initForm: () =>
		new MarketplaceItemStepForm({
			collection: 'custom_checks',
			entityData: entities.custom_checks
		}),
	serialize: (item) => ({ check_id: getPath(item) }),
	deserialize: async ({ check_id }) => {
		if (!check_id) throw new Error('Missing check_id');
		return getMarketplaceItemByPath(check_id);
	},
	cardData: getMarketplaceItemCardData,
	makeId: ({ check_id }) => getLastPathSegment(check_id ?? 'custom-check-unknown')
};

//

function getMarketplaceItemCardData(item: MarketplaceItem) {
	return {
		title: item.name,
		copyText: item.path,
		avatar: getMarketplaceItemLogo(item)
	};
}
