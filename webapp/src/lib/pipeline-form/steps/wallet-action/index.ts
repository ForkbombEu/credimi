// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';
import { entities } from '$lib/global/entities';
import {
	getMarketplaceItemLogo,
	getMarketplaceItemUrl,
	type MarketplaceItem
} from '$lib/marketplace';
import { type PipelineStepByType, type PipelineStepData } from '$lib/pipeline/types.js';
import { getPath } from '$lib/utils';

import { m } from '@/i18n/index.js';
import { pb } from '@/pocketbase';
import {
	type MobileRunnersResponse,
	type WalletActionsResponse,
	type WalletVersionsResponse
} from '@/pocketbase/types';

import type { TypedConfig } from '../types';

import { getLastPathSegment } from '../_partials/misc';
import CardDetailsComponent from './card-details.svelte';
import EditComponent from './edit-component.svelte';
import {
	WalletActionStepForm,
	type WalletActionStepData
} from './wallet-action-step-form.svelte.js';

//

export const walletActionStepConfig: TypedConfig<'mobile-automation', WalletActionStepData> = {
	use: 'mobile-automation',

	display: entities.wallets,

	CardDetailsComponent,
	EditComponent,

	cardData: ({ action, wallet, version, runner }) => {
		let publicUrl = getMarketplaceItemUrl(wallet);
		publicUrl += `#${action.canonified_name}`;
		return {
			title: action.name,
			copyText: getPath(action),
			avatar: getMarketplaceItemLogo(wallet),
			publicUrl,
			meta: {
				wallet: `${wallet.name} (v. ${version.tag})`,
				runner: runner === 'global' ? m.Choose_later() : runner.name
			}
		};
	},

	makeId: (data) => {
		if (!('action_id' in data) || !('version_id' in data)) {
			console.log(data);
			throw new Error('Invalid data');
		}
		return getLastPathSegment(data.action_id);
	},

	initForm: () => new WalletActionStepForm(),

	serialize: ({ action, version, runner }) => {
		const _with: PipelineStepData<PipelineStepByType<'mobile-automation'>> = {
			action_id: getPath(action),
			version_id: getPath(version)
		};
		if (runner !== 'global') {
			_with.runner_id = getPath(runner);
		}
		if (action.code.includes('${DL}') || action.code.includes('${deeplink}')) {
			_with.parameters = {
				deeplink: '<deeplink-placeholder>' // will be written later
			};
		}
		return _with;
	},

	deserialize: async (data) => {
		if (!('action_id' in data) || !('version_id' in data)) {
			throw new Error('Invalid data');
		}

		const action = await getRecordByCanonifiedPath<WalletActionsResponse>(data.action_id);
		const version = await getRecordByCanonifiedPath<WalletVersionsResponse>(data.version_id);
		if (isError(action) || isError(version)) {
			throw new Error('Failed to get record by canonified path');
		}

		let runner: WalletActionStepData['runner'] = 'global';
		if (data.runner_id) {
			const response = await getRecordByCanonifiedPath<MobileRunnersResponse>(data.runner_id);
			if (!isError(response)) runner = response;
		}

		const wallet: MarketplaceItem = await pb
			.collection('marketplace_items')
			.getOne(action.wallet);

		return { wallet, version, action, runner };
	}
};

//

function isError(value: unknown): value is Error {
	return value instanceof Error;
}
