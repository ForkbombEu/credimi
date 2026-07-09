// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { TypedConfig } from '$pipeline-form/steps/types';

import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';
import { entities } from '$lib/global/entities.js';
import { getCustomCheckPublicUrl } from '$lib/hub/utils.js';
import { getPath } from '$lib/utils';
import { getLastPathSegment } from '$pipeline-form/steps/_partials/index.js';

import type { CustomChecksResponse } from '@/pocketbase/types';

import { localizeHref, m } from '@/i18n/index.js';
import { pb } from '@/pocketbase';

import {
	CustomIntegrationStepForm,
	type CustomIntegrationStepFormData
} from './custom-integration-step-form.svelte.js';

export type { CustomIntegrationStepFormData };

export const customCheckStepConfig: TypedConfig<'custom-check', CustomIntegrationStepFormData> = {
	use: 'custom-check',
	display: entities.custom_checks,
	initForm: (opts) => new CustomIntegrationStepForm(opts),
	serialize: ({ integration, parameters }) => {
		const serialized: { check_id: string; parameters?: Record<string, unknown> } = {
			check_id: getPath(integration, true)
		};
		if (parameters && Object.keys(parameters).length > 0) {
			serialized.parameters = parameters;
		}
		return serialized;
	},
	deserialize: async ({ check_id, parameters }) => {
		if (!check_id) throw new Error(m.Pipeline_form_missing_check_id());
		const integration = await getRecordByCanonifiedPath<CustomChecksResponse>(check_id);
		if (integration instanceof Error) throw integration;
		return {
			integration,
			parameters: parameters as Record<string, unknown> | undefined
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
