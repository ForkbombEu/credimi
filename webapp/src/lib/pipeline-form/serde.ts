// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getMarketplaceItemData, type MarketplaceItem } from '$lib/marketplace/utils.js';
import { getPath } from '$lib/utils/index.js';
import { parse } from 'yaml';

import { pb } from '@/pocketbase';
import { Collections } from '@/pocketbase/types/index.js';

import type { Metadata } from './metadata-form/metadata-form.svelte.js';
import type { ActivityOptions, Pipeline } from './types';

import { isActivityOptions } from './activity-options-form/activity-options-form.svelte.js';
import { metadataSchema } from './metadata-form/metadata-form.svelte.js';
import { StepType, type BuilderStep } from './steps-builder/types.js';

//

export type PipelineData = {
	id: string;
	metadata: Metadata;
	steps: BuilderStep[];
	activityOptions: ActivityOptions;
	yaml: string;
	description: string;
};

export async function fetchPipeline(id: string, options = { fetch }): Promise<PipelineData> {
	const baseData = await pb
		.collection('pipelines')
		.getOne(id, { fetch: options.fetch, requestKey: null });

	const metadata = metadataSchema.parse(baseData);

	const activityOptions = (parse(baseData.yaml) as Pipeline)?.runtime?.temporal?.activity_options;
	if (!isActivityOptions(activityOptions)) {
		throw new Error('Invalid activity options');
	}

	const steps = baseData.steps as SerializedStep[];
	const deserializedSteps = await Promise.all(steps.map((s) => deserializeStep(s, options)));

	return {
		id,
		metadata,
		activityOptions,
		steps: deserializedSteps,
		yaml: baseData.yaml,
		description: baseData.description
	};
}

//

export function serializeStep(step: BuilderStep) {
	const continueOnError = step.continueOnError ?? false;
	if (step.type === StepType.WalletAction) {
		return {
			id: step.id,
			type: step.type,
			actionId: step.data.action.id,
			walletId: step.data.wallet.id,
			versionId: step.data.version.id,
			continueOnError,
			video: step.video ?? false
		};
	} else if (step.type === StepType.ConformanceCheck) {
		return {
			id: step.id,
			type: step.type,
			checkId: step.data.checkId,
			continueOnError
		};
	} else {
		return {
			id: step.id,
			type: step.type,
			recordId: step.data.id,
			continueOnError
		};
	}
}

type SerializedStep = ReturnType<typeof serializeStep>;

async function deserializeStep(step: SerializedStep, options = { fetch }): Promise<BuilderStep> {
	if (step.type === StepType.WalletAction) {
		const action = await pb.collection('wallet_actions').getOne(step.actionId, {
			fetch: options.fetch,
			requestKey: null
		});
		const walletItem: MarketplaceItem = await pb
			.collection('marketplace_items')
			.getFirstListItem(`type = "${Collections.Wallets}" && id = "${action.wallet}"`, {
				fetch: options.fetch,
				requestKey: null
			});
		const version = await pb.collection('wallet_versions').getOne(step.versionId, {
			fetch: options.fetch,
			requestKey: null
		});
		const avatar = getMarketplaceItemData(walletItem).logo;
		return {
			type: StepType.WalletAction,
			id: step.id,
			name: action.name,
			path: getPath(action),
			organization: walletItem.organization_name,
			avatar: avatar,
			continueOnError: step.continueOnError ?? false,
			video: step.video ?? false,
			data: {
				wallet: walletItem,
				version: version,
				action: action
			}
		};
	} else if (step.type === StepType.ConformanceCheck) {
		return {
			type: StepType.ConformanceCheck,
			id: step.id,
			name: step.checkId,
			path: step.checkId,
			organization: 'Conformance Check',
			data: { checkId: step.checkId },
			continueOnError: step.continueOnError ?? false,
			video: step.video ?? false
		};
	} else {
		const record: MarketplaceItem = await pb
			.collection('marketplace_items')
			.getOne(step.recordId, {
				fetch: options.fetch,
				requestKey: null
			});
		const avatar = getMarketplaceItemData(record).logo;
		return {
			type: step.type,
			id: step.id,
			name: record.name,
			path: getPath(record),
			organization: record.organization_name,
			avatar: avatar,
			data: record,
			continueOnError: step.continueOnError ?? false,
			video: step.video ?? false
		};
	}
}
