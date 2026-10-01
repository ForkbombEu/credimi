// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Record } from '$lib/pipeline/device';
import type { TypedConfig } from '$pipeline-form/steps/types';

import { Wallet } from '$lib';
import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';
import { entities } from '$lib/global/entities';
import { getHubItemById, getHubItemLogo, getHubItemUrl } from '$lib/hub';
import { resolveByPath } from '$lib/pipeline/device/query.js';
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
import { type WalletActionsResponse, type WalletVersionsResponse } from '@/pocketbase/types';

import type {
	InlineWalletActionStepData,
	StoredWalletActionStepData,
	WalletActionStepData
} from './types.js';

import CardDetailsComponent from './card-details.svelte';
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
export { InlineWalletActionStepForm } from './inline-wallet-action-step-form.svelte.js';
export { WalletActionStepForm } from './wallet-action-step-form.svelte.js';

//

type MobileWith = PipelineStepData<PipelineStepByType<'mobile-automation'>>;

async function resolveVersion(versionId: string): Promise<SelectedVersion> {
	if (versionId === EXTERNAL_VERSION) return EXTERNAL_VERSION;
	const response = await getRecordByCanonifiedPath<WalletVersionsResponse>(versionId);
	if (!isError(response)) return response;
	throw response;
}

function fallbackDevice(path: string): Record {
	return {
		name: getLastPathSegment(path),
		path,
		isOwned: false,
		isPublished: false,
		isOnline: false
	};
}

async function resolveDevice(deviceId: string | undefined): Promise<SelectedDevice> {
	if (!deviceId || deviceId === GLOBAL_DEVICE) return GLOBAL_DEVICE;

	const path = deviceId;
	try {
		return (await resolveByPath(path)) ?? fallbackDevice(path);
	} catch {
		return fallbackDevice(path);
	}
}

function parametersFrom(data: MobileWith): { [key: string]: string } | undefined {
	if (!('parameters' in data) || !data.parameters) return undefined;
	return data.parameters as { [key: string]: string };
}

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

	makeId: (data) => {
		if ('action_code' in data && typeof data.action_code === 'string') {
			return 'inline-maestro';
		}
		if (!('action_id' in data) || !('version_id' in data)) {
			throw new Error(m.Pipeline_form_invalid_step_data());
		}
		return getLastPathSegment(data.action_id);
	},

	initForm: (opts) => {
		if (opts?.initial && isInlineWalletActionStepData(opts.initial)) {
			return new InlineWalletActionStepForm(opts);
		}
		return new WalletActionStepForm(opts);
	},

	serialize: (data) => {
		if (data.kind === 'inline') {
			const _with: MobileWith = {
				action_code: data.actionCode,
				version_id:
					data.version === EXTERNAL_VERSION ? EXTERNAL_VERSION : getPath(data.version)
			};
			if (data.device !== GLOBAL_DEVICE) {
				_with.device_id = data.device.path;
			}
			if (data.parameters && Object.keys(data.parameters).length > 0) {
				_with.parameters = data.parameters;
			}
			return _with;
		}

		const { action, version, device, parameters } = data;
		const _with: MobileWith = {
			action_id: getPath(action),
			version_id: version === EXTERNAL_VERSION ? EXTERNAL_VERSION : getPath(version)
		};
		if (device !== GLOBAL_DEVICE) {
			_with.device_id = device.path;
		}
		if (parameters && Object.keys(parameters).length > 0) {
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
		if ('action_code' in data && typeof data.action_code === 'string' && 'version_id' in data) {
			const version = await resolveVersion(String(data.version_id));
			const device = await resolveDevice(
				'device_id' in data ? (data.device_id as string | undefined) : undefined
			);
			return {
				kind: 'inline',
				actionCode: data.action_code,
				version,
				device,
				parameters: parametersFrom(data)
			} satisfies InlineWalletActionStepData;
		}

		if (!('action_id' in data) || !('version_id' in data)) {
			throw new Error(m.Pipeline_form_invalid_step_data());
		}

		const action = await getRecordByCanonifiedPath<WalletActionsResponse>(data.action_id);
		if (isError(action)) {
			throw action;
		}

		const version = await resolveVersion(String(data.version_id));
		const device = await resolveDevice(
			'device_id' in data ? (data.device_id as string | undefined) : undefined
		);
		const wallet = await getHubItemById(action.wallet);

		return {
			kind: 'stored',
			wallet,
			version,
			action,
			device,
			parameters: parametersFrom(data)
		} satisfies StoredWalletActionStepData;
	}
};
