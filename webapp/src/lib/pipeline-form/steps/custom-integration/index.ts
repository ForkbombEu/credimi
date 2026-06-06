// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';
import { entities } from '$lib/global/entities.js';
import { getCustomCheckPublicUrl } from '$lib/hub/utils.js';
import { getPath } from '$lib/utils';

import type { CustomChecksResponse } from '@/pocketbase/types';

import { localizeHref, m } from '@/i18n/index.js';
import { pb } from '@/pocketbase';

import type { TypedConfig } from '../types';

import { getLastPathSegment } from '../_partials/misc';
import {
	CustomIntegrationStepForm,
	type CustomIntegrationStepFormData
} from './custom-integration-step-form.svelte.js';

export type { CustomIntegrationStepFormData };

export const customCheckStepConfig: TypedConfig<'custom-check', CustomIntegrationStepFormData> = {
	use: 'custom-check',
	display: entities.custom_checks,
	initForm: (opts) => new CustomIntegrationStepForm(opts),
	serialize: ({ integration, config }) => {
		const serialized: { check_id: string; config?: Record<string, unknown> } = {
			check_id: getPath(integration, true)
		};
		if (config && Object.keys(config).length > 0) {
			serialized.config = config;
		}
		return serialized;
	},
	deserialize: async ({ check_id, config }) => {
		if (!check_id) throw new Error(m.Pipeline_form_missing_check_id());
		const integration = await getRecordByCanonifiedPath<CustomChecksResponse>(check_id);
		if (integration instanceof Error) throw integration;
		return {
			integration,
			config: config as Record<string, unknown> | undefined
		};
	},
	cardData: ({ integration }) => ({
		title: integration.name,
		copyText: getPath(integration, true),
		avatar: integration.logo ? pb.files.getURL(integration, integration.logo) : undefined,
		publicUrl: localizeHref(getCustomCheckPublicUrl(integration))
	}),
	makeId: ({ check_id }) => getLastPathSegment(check_id ?? 'custom-check-unknown')
};
