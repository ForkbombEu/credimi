// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$lib/marketplace/utils.js';

import { getPath } from '$lib/utils/index.js';
import { parse } from 'yaml';

import { pb } from '@/pocketbase';
import { Collections } from '@/pocketbase/types/index.js';

import type { Metadata } from './metadata-form/metadata-form.svelte.js';
import type { ActivityOptions } from './types.generated.js';

import { isActivityOptions } from './activity-options-form/activity-options-form.svelte.js';
import { metadataSchema } from './metadata-form/metadata-form.svelte.js';
import { StepType, type BuilderStep } from './steps-builder/types.js';

//

export type PipelineData = {
	metadata: Metadata;
	steps: BuilderStep[];
	activityOptions: ActivityOptions;
};

export async function fetchPipeline(id: string, options = { fetch }): Promise<PipelineData> {
	const baseData = await pb
		.collection('pipelines')
		.getOne(id, { fetch: options.fetch, requestKey: null });

	const metadata = metadataSchema.parse(baseData);

	const activityOptions = parse(baseData.yaml);
	if (!isActivityOptions(activityOptions)) {
		throw new Error('Invalid activity options');
	}

	const steps = JSON.parse(baseData.steps as string) as SerializedStep[];

	return {
		metadata,
		activityOptions,
		steps: await Promise.all(steps.map(deserializeStep))
	};
}

//

export function serializeStep(step: BuilderStep) {
	const continueOnError = step.continueOnError ?? false;
	if (step.type === StepType.WalletAction) {
		return {
			type: step.type,
			actionId: step.data.action.id,
			walletId: step.data.wallet.id,
			versionId: step.data.version.id,
			continueOnError
		};
	} else {
		return {
			type: step.type,
			recordId: step.data.id,
			continueOnError
		};
	}
}

type SerializedStep = ReturnType<typeof serializeStep>;

async function deserializeStep(step: SerializedStep): Promise<BuilderStep> {
	if (step.type === StepType.WalletAction) {
		const action = await pb.collection('wallet_actions').getOne(step.actionId);
		const walletItem: MarketplaceItem = await pb
			.collection('marketplace_items')
			.getFirstListItem(`type = "${Collections.Wallets}" && id = "${action.wallet}"`);
		const version = await pb.collection('wallet_versions').getOne(step.versionId);
		return {
			type: StepType.WalletAction,
			id: action.id,
			name: action.name,
			path: getPath(action),
			organization: walletItem.organization_name,
			data: {
				wallet: walletItem,
				version: version,
				action: action
			}
		};
	} else {
		const record: MarketplaceItem = await pb
			.collection('marketplace_items')
			.getOne(step.recordId);
		return {
			type: step.type,
			id: record.id,
			name: record.name,
			path: getPath(record),
			organization: record.organization_name,
			data: record
		};
	}
}
