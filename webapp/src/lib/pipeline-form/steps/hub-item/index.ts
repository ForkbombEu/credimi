// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub/types.js';

import { entities } from '$lib/global/entities.js';
import { getHubItemByPath, getHubItemLogo, getHubItemUrl } from '$lib/hub/utils.js';
import { getPath } from '$lib/utils';

import { m } from '@/i18n/index.js';

import type { TypedConfig } from '../types';

import { getLastPathSegment } from '../_partials/misc';
import { HubItemStepForm } from './hub-item-step-form.svelte.js';

//

export const credentialsStepConfig: TypedConfig<'credential-offer', HubItem> = {
	use: 'credential-offer',
	display: {
		...entities.credentials,
		labels: { ...entities.credentials.labels, singular: m.Credential_Deeplink() }
	},
	initForm: (opts) =>
		new HubItemStepForm(
			{
				collection: 'credentials',
				entityData: entities.credentials
			},
			opts
		),
	serialize: (item) => ({ credential_id: getPath(item) }),
	deserialize: ({ credential_id }) => getHubItemByPath(credential_id),
	cardData: getHubItemCardData,
	makeId: ({ credential_id }) => getLastPathSegment(credential_id)
};

//

export const useCaseVerificationStepConfig: TypedConfig<'use-case-verification-deeplink', HubItem> =
	{
		use: 'use-case-verification-deeplink',
		display: entities.use_cases_verifications,
		initForm: (opts) =>
			new HubItemStepForm(
				{
					collection: 'use_cases_verifications',
					entityData: entities.use_cases_verifications
				},
				opts
			),
		serialize: (item) => ({ use_case_id: getPath(item) }),
		deserialize: ({ use_case_id }) => getHubItemByPath(use_case_id),
		cardData: getHubItemCardData,
		makeId: ({ use_case_id }) => getLastPathSegment(use_case_id)
	};

//

export const customCheckStepConfig: TypedConfig<'custom-check', HubItem> = {
	use: 'custom-check',
	display: entities.custom_checks,
	initForm: (opts) =>
		new HubItemStepForm(
			{
				collection: 'custom_checks',
				entityData: entities.custom_checks
			},
			opts
		),
	serialize: (item) => ({ check_id: getPath(item) }),
	deserialize: async ({ check_id }) => {
		if (!check_id) throw new Error('Missing check_id');
		return getHubItemByPath(check_id);
	},
	cardData: getHubItemCardData,
	makeId: ({ check_id }) => getLastPathSegment(check_id ?? 'custom-check-unknown')
};

//

function getHubItemCardData(item: HubItem) {
	return {
		title: item.name,
		copyText: getPath(item),
		avatar: getHubItemLogo(item),
		publicUrl: getHubItemUrl(item)
	};
}
