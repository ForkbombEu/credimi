// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities } from '$lib/global/entities';
import { getMarketplaceItemLogo, type MarketplaceItem } from '$lib/marketplace';
import {
	DEEPLINK_STEP_ID_PLACEHOLDER,
	type PipelineStepByType,
	type PipelineStepData,
	type TypedPipelineStepConfig
} from '$lib/pipeline-form/types';
import { getPath } from '$lib/utils';

import { pb } from '@/pocketbase';
import { Collections } from '@/pocketbase/types';

import {
	WalletActionStepForm,
	type WalletActionStepData
} from './wallet-action-step-form.svelte.js';

//

export const walletActionStepConfig: TypedPipelineStepConfig<
	'mobile-automation',
	WalletActionStepData
> = {
	id: 'mobile-automation',

	display: entities.wallets,
	cardData: ({ action, wallet }) => ({
		title: action.name,
		copyText: getPath(action),
		avatar: getMarketplaceItemLogo(wallet)
	}),

	initForm: () => new WalletActionStepForm(),

	serialize: ({ action, version }) => {
		const _with: PipelineStepData<PipelineStepByType<'mobile-automation'>> = {
			action_id: getPath(action),
			version_id: getPath(version)
		};
		if (action.code.includes('${DL}') || action.code.includes('${deeplink}')) {
			_with.parameters = {
				deeplink: '${{' + DEEPLINK_STEP_ID_PLACEHOLDER + '}}'
			};
		}
		return _with;
	},

	deserialize: async (data) => {
		if (!('action_id' in data) || !('version_id' in data)) {
			throw new Error('Invalid data');
		}
		const [orgId, walletId, actionId] = data.action_id.split('/').filter(Boolean);
		const versionId = data.version_id.split('/').filter(Boolean)[2];

		const wallet: MarketplaceItem = await pb.collection('marketplace_items').getFirstListItem(
			pb.filter('type = {:type} && canonified_name = {:walletId}', {
				type: Collections.Wallets,
				walletId
			})
		);

		const action = await pb.collection('wallet_actions').getFirstListItem(
			pb.filter(
				[
					'owner.canonified_name={:orgId}',
					'wallet.canonified_name={:walletId}',
					'canonified_name={:actionId}'
				].join('&&'),
				{
					orgId,
					walletId,
					actionId
				}
			)
		);

		const version = await pb.collection('wallet_versions').getFirstListItem(
			pb.filter(
				[
					'owner.canonified_name = {:orgId}',
					'wallet.canonified_name = {:walletId}',
					'canonified_tag = {:versionId}'
				].join('&&'),
				{
					orgId,
					walletId,
					versionId
				}
			)
		);

		return {
			wallet: wallet,
			version: version,
			action: action
		};
	}
};
