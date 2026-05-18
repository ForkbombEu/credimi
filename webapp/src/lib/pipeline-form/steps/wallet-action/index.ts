// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Wallet } from '$lib';
import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';
import { entities } from '$lib/global/entities';
import { getHubItemLogo, getHubItemUrl, type HubItem } from '$lib/hub';
import { fetchAvailableRunners, type MobileRunnerListItem } from '$lib/pipeline/runners/utils';
import {
	type PipelineStepByType,
	type PipelineStepData,
	type PipelineStepType
} from '$lib/pipeline/types.js';
import { getPath } from '$lib/utils';

import { m } from '@/i18n/index.js';
import { pb } from '@/pocketbase';
import { type WalletActionsResponse, type WalletVersionsResponse } from '@/pocketbase/types';

import type { TypedConfig } from '../types';

import { getLastPathSegment } from '../_partials/misc';
import { formatLinkedId } from '../utils.js';
import CardDetailsComponent from './card-details.svelte';
import EditComponent from './edit-component.svelte';
import {
	EXTERNAL_VERSION,
	getRunnerLabel,
	getVersionLabel,
	GLOBAL_RUNNER,
	WalletActionStepForm,
	type SelectedRunner,
	type SelectedVersion,
	type WalletActionStepData
} from './wallet-action-step-form.svelte.js';

//

export const walletActionStepConfig: TypedConfig<'mobile-automation', WalletActionStepData> = {
	use: 'mobile-automation',

	display: entities.wallets,

	CardDetailsComponent,
	EditComponent,

	cardData: ({ action, wallet, version, runner }) => {
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
				[m.Runner()]: getRunnerLabel(runner),
				[m.Version()]: getVersionLabel(version)
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
		type StepData = PipelineStepData<PipelineStepByType<'mobile-automation'>>;
		const _with: StepData = {
			action_id: getPath(action),
			version_id: version === EXTERNAL_VERSION ? EXTERNAL_VERSION : getPath(version)
		};
		if (runner !== GLOBAL_RUNNER) {
			_with.runner_id = runner.runner_id;
		}
		if (action.code.includes('${DL}') || action.code.includes('${deeplink}')) {
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
			throw new Error('Invalid data');
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

		let runner: SelectedRunner = GLOBAL_RUNNER;
		if (data.runner_id !== GLOBAL_RUNNER && data.runner_id) {
			const fallbackRunner = {
				mine: false,
				name: getLastPathSegment(data.runner_id),
				online: false,
				published: false,
				runner_id: data.runner_id
			} satisfies MobileRunnerListItem;

			try {
				runner =
					(await fetchAvailableRunners()).find(
						(item) => item.runner_id === data.runner_id
					) ?? fallbackRunner;
			} catch {
				runner = fallbackRunner;
			}
		}

		const wallet: HubItem = await pb.collection('hub_items').getOne(action.wallet);

		return { wallet, version, action, runner };
	}
};

//

function isError(value: unknown): value is Error {
	return value instanceof Error;
}
