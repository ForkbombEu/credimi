// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Record } from '$lib/pipeline/device';
import type { TypedConfig } from '$pipeline-form/steps/types';

import { Pipeline, Wallet } from '$lib';
import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';
import { entities } from '$lib/global/entities';
import { getHubItemLogo, getHubItemUrl, type HubItem } from '$lib/hub';
import {
	type PipelineStepByType,
	type PipelineStepData,
	type PipelineStepType
} from '$lib/pipeline/types.js';
import { getPath } from '$lib/utils';
import {
	EXTERNAL_VERSION,
	GLOBAL_DEVICE,
	type SelectedDevice,
	type SelectedVersion
} from '$pipeline-form/execution-target/types.js';
import { getLastPathSegment } from '$pipeline-form/steps/_partials/index.js';
import { formatLinkedId } from '$pipeline-form/steps/utils.js';
import { isError } from 'effect/Predicate';

import { m } from '@/i18n/index.js';
import { pb } from '@/pocketbase';
import { type WalletActionsResponse, type WalletVersionsResponse } from '@/pocketbase/types';

import type { WalletActionStepData } from './types.js';

import CardDetailsComponent from './card-details.svelte';
import {
	getDeviceLabel,
	getVersionLabel,
	WalletActionStepForm
} from './wallet-action-step-form.svelte.js';

export type { WalletActionStepData } from './types.js';
export { isWalletActionStepData } from './types.js';

//

export const walletActionStepConfig: TypedConfig<'mobile-automation', WalletActionStepData> = {
	use: 'mobile-automation',

	display: entities.wallets,

	CardDetailsComponent,

	cardData: ({ action, wallet, version, device }) => {
		let publicUrl = getHubItemUrl(wallet);
		publicUrl += `#${action.canonified_name}`;

		return {
			title: action.name,
			copyText: getPath(action),
			avatar: getHubItemLogo(wallet),
			publicUrl,
			beforeTitle: Wallet.Action.getCategoryLabel(action),
			meta: {
				[m.Wallet()]: wallet.name,
				['Device']: getDeviceLabel(device),
				[m.Version()]: getVersionLabel(version)
			}
		};
	},

	makeId: (data) => {
		if (!('action_id' in data) || !('version_id' in data)) {
			throw new Error(m.Pipeline_form_invalid_step_data());
		}
		return getLastPathSegment(data.action_id);
	},

	initForm: (opts) => new WalletActionStepForm(opts),

	serialize: ({ action, version, device, parameters }) => {
		type StepData = PipelineStepData<PipelineStepByType<'mobile-automation'>>;
		const _with: StepData = {
			action_id: getPath(action),
			version_id: version === EXTERNAL_VERSION ? EXTERNAL_VERSION : getPath(version)
		};
		if (device !== GLOBAL_DEVICE) {
			_with.device_id = device.path;
		}
		if (parameters && Object.keys(parameters).length > 0) {
			// Keep the exact bound parameters so editing a step never drops
			// values like deeplink references or custom action variables.
			_with.parameters = parameters;
		} else if (action.code.includes('${DL}') || action.code.includes('${deeplink}')) {
			_with.parameters = {
				deeplink: '<deeplink-placeholder>'
			};
		}
		return _with;
	},

	linkProcedure: (serialized, previousSteps) => {
		if (!serialized.parameters?.deeplink) return;

		const linkableSteps: PipelineStepType[] = [
			'conformance-check',
			'credential-offer',
			'use-case-verification-deeplink',
			'custom-check'
		];
		const previousStep = previousSteps
			.toReversed()
			.filter((s) => linkableSteps.includes(s.use))
			.at(0);

		if (!previousStep) return;
		serialized.parameters.deeplink = formatLinkedId(previousStep);
	},

	deserialize: async (data) => {
		if (!('action_id' in data) || !('version_id' in data)) {
			throw new Error(m.Pipeline_form_invalid_step_data());
		}

		const action = await getRecordByCanonifiedPath<WalletActionsResponse>(data.action_id);
		if (isError(action)) {
			throw action;
		}

		let version: SelectedVersion = EXTERNAL_VERSION;
		if (data.version_id !== EXTERNAL_VERSION) {
			const response = await getRecordByCanonifiedPath<WalletVersionsResponse>(
				data.version_id
			);
			if (!isError(response)) {
				version = response;
			} else {
				throw response;
			}
		}

		let device: SelectedDevice = GLOBAL_DEVICE;
		if (data.device_id !== GLOBAL_DEVICE && data.device_id) {
			const path = data.device_id;
			const fallbackDevice = {
				name: getLastPathSegment(path),
				path,
				isOwned: false,
				isPublished: false,
				isOnline: false
			} satisfies Record;

			await Pipeline.Device.fetchRecords().match({
				Rejected: () => {
					device = fallbackDevice;
				},
				Resolved: (devices) => {
					device = devices.find((item) => item.path === path) ?? fallbackDevice;
				}
			});
		}

		const wallet: HubItem = await pb.collection('hub_items').getOne(action.wallet);

		return {
			wallet,
			version,
			action,
			device,
			parameters: data.parameters as { [key: string]: string } | undefined
		};
	}
};
