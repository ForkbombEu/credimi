// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$lib/marketplace/types.js';
import type { TypedPipelineStepConfig } from '$lib/pipeline-form/types';

import { entities } from '$lib/global/entities.js';
import { getMarketplaceItemByPath } from '$lib/marketplace/utils.js';

import { m } from '@/i18n/index.js';

import { MarketplaceItemStepForm } from './marketplace-item-step-form.svelte.js';

//

export const credentialsStepConfig: TypedPipelineStepConfig<'credential-offer', MarketplaceItem> = {
	id: 'credential-offer',
	display: {
		...entities.credentials,
		labels: { ...entities.credentials.labels, singular: m.Credential_Deeplink() }
	},
	initForm: () =>
		new MarketplaceItemStepForm({
			collection: 'credentials',
			entityData: entities.credentials
		}),
	serialize: (item) => ({ credential_id: item.path }),
	deserialize: ({ credential_id }) => getMarketplaceItemByPath(credential_id)
};

//

export const useCaseVerificationStepConfig: TypedPipelineStepConfig<
	'use-case-verification-deeplink',
	MarketplaceItem
> = {
	id: 'use-case-verification-deeplink',
	display: entities.use_cases_verifications,
	initForm: () =>
		new MarketplaceItemStepForm({
			collection: 'use_cases_verifications',
			entityData: entities.use_cases_verifications
		}),
	serialize: (item) => ({ use_case_id: item.path }),
	deserialize: ({ use_case_id }) => getMarketplaceItemByPath(use_case_id)
};

//

export const customCheckStepConfig: TypedPipelineStepConfig<'custom-check', MarketplaceItem> = {
	id: 'custom-check',
	display: entities.custom_checks,
	initForm: () =>
		new MarketplaceItemStepForm({
			collection: 'custom_checks',
			entityData: entities.custom_checks
		}),
	serialize: (item) => ({ check_id: item.path }),
	deserialize: async ({ check_id }) => {
		if (!check_id) throw new Error('Missing check_id');
		return getMarketplaceItemByPath(check_id);
	}
};
