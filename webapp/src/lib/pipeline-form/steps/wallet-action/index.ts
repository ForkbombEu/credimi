// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { TypedConfig } from '$pipeline-form/steps/types';

import { Wallet } from '$lib';
import { entities } from '$lib/global/entities';
import { getHubItemLogo, getHubItemUrl } from '$lib/hub';
import { type PipelineStepType } from '$lib/pipeline/types.js';
import { getPath } from '$lib/utils';
import { formatLinkedId } from '$pipeline-form/steps/utils.js';

import { m } from '@/i18n/index.js';

import type { WalletActionStepData } from './types.js';

import CardDetailsComponent from './card-details.svelte';
import { deserializeWalletAction, makeWalletActionId, serializeWalletAction } from './codec.js';
import { InlineWalletActionStepForm } from './inline-wallet-action-step-form.svelte.js';
import { getDeviceLabel, getVersionLabel } from './labels.js';
import { isInlineWalletActionStepData } from './types.js';
import { WalletActionStepForm } from './wallet-action-step-form.svelte.js';

export type {
	InlineWalletActionStepData,
	StoredWalletActionStepData,
	WalletActionStepData
} from './types.js';
export { isInlineWalletActionStepData, isStoredWalletActionStepData } from './types.js';
export { applyWalletActionStepVersion, isMatchingMobileStepData } from './change-wallet-version.js';
export { InlineWalletActionStepForm } from './inline-wallet-action-step-form.svelte.js';
export { WalletActionStepForm } from './wallet-action-step-form.svelte.js';

//

export const walletActionStepConfig: TypedConfig<'mobile-automation', WalletActionStepData> = {
	use: 'mobile-automation',

	display: entities.wallets,

	CardDetailsComponent,

	cardData: (data) => {
		if (data.kind === 'inline') {
			return {
				title: m.Custom_wallet_action(),
				beforeTitle: m.Inline_Maestro(),
				meta: {
					Code: m.Custom_code(),
					['Device']: getDeviceLabel(data.device),
					[m.Version()]: getVersionLabel(data.version)
				}
			};
		}

		const { action, wallet, version, device } = data;
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

	makeId: makeWalletActionId,

	initForm: (opts) => {
		if (opts?.initial && isInlineWalletActionStepData(opts.initial)) {
			return new InlineWalletActionStepForm(opts);
		}
		return new WalletActionStepForm(opts);
	},

	serialize: serializeWalletAction,

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

	deserialize: deserializeWalletAction
};
