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
			entityData: entities.credentials,
			onSelect: (item) => ({
				use: 'credential-offer',
				id: item.path,
				with: { credential_id: item.path }
			})
		}),
	serialize: (item) => ({ credential_id: item.path }),
	deserialize: async ({ credential_id }) => {
		return getMarketplaceItemByPath(credential_id);
	}
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
			entityData: entities.use_cases_verifications,
			onSelect: (item) => ({
				use: 'use-case-verification-deeplink',
				id: item.path,
				with: { use_case_id: item.path }
			})
		}),
	serialize: (item) => ({ use_case_id: item.path }),
	deserialize: async ({ use_case_id }) => {
		return getMarketplaceItemByPath(use_case_id);
	}
};

//

export const customCheckStepConfig: TypedPipelineStepConfig<'custom-check', MarketplaceItem> = {
	id: 'custom-check',
	display: entities.custom_checks,
	initForm: () =>
		new MarketplaceItemStepForm({
			collection: 'custom_checks',
			entityData: entities.custom_checks,
			onSelect: (item) => ({
				use: 'custom-check',
				id: item.path,
				with: {
					check_id: item.path
				}
			})
		}),
	serialize: (item) => ({ custom_check_id: item.path }),
	deserialize: async (data) => {
		return getMarketplaceItemByPath(data['with'] as string);
	}
};
