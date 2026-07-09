// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';
import type { PipelineStepByType } from '$lib/pipeline/types';
import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';

import { isError } from 'effect/Predicate';

import { isWalletActionStepData } from '../../steps/wallet-action/types.js';

//

export type BulkWalletVersionContext = {
	wallet: HubItem;
	versionId: string;
	mobileIndices: number[];
};

function mobileWith(step: EnrichedStep[0]): PipelineStepByType<'mobile-automation'>['with'] | null {
	if (step.use !== 'mobile-automation' || !('with' in step)) return null;
	return step.with as PipelineStepByType<'mobile-automation'>['with'];
}

/**
 * When non-null, the pipeline has at least one mobile-automation step, every such step
 * is successfully enriched, and all share the same wallet and serialized version_id.
 */
export function getBulkWalletVersionContext(
	steps: EnrichedStep[]
): BulkWalletVersionContext | null {
	const mobileIndices: number[] = [];
	for (let i = 0; i < steps.length; i++) {
		const [raw] = steps[i]!;
		if (raw.use === 'mobile-automation') mobileIndices.push(i);
	}
	if (mobileIndices.length === 0) return null;

	let walletId: string | undefined;
	let versionId: string | undefined;
	let wallet: HubItem | undefined;

	for (const i of mobileIndices) {
		const tuple = steps[i]!;
		const [, data] = tuple;
		if (isError(data)) return null;
		if (!isWalletActionStepData(data)) return null;

		const w = mobileWith(tuple[0]);
		if (!w || !('version_id' in w) || typeof w.version_id !== 'string') return null;

		if (walletId === undefined) {
			walletId = data.wallet.id;
			versionId = w.version_id;
			wallet = data.wallet;
		} else if (data.wallet.id !== walletId || w.version_id !== versionId) {
			return null;
		}
	}

	if (!wallet || versionId === undefined) return null;

	return { wallet, versionId, mobileIndices };
}
