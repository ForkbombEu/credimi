// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineStepByType } from '$lib/pipeline/types.js';

import { isError } from 'effect/Predicate';

import type { GenericRecord } from '@/utils/types';

import type { EnrichedStep } from '../shared/enriched-step.js';
import type { SelectedVersion } from '../shared/mobile-target.js';
import type { WalletActionStepData } from '../steps/wallet-action/types.js';

import { walletActionStepConfig } from '../steps/wallet-action/index.js';

export function syncMobileStepVersionsIfSameWallet(
	steps: EnrichedStep[],
	walletId: string,
	version: SelectedVersion
): EnrichedStep[] {
	return steps.map((tuple) => {
		const [raw, data] = tuple;
		if (raw.use !== 'mobile-automation') return tuple;
		if (isError(data)) return tuple;

		const stepData = data as unknown as WalletActionStepData;
		if (stepData.wallet.id !== walletId) return tuple;

		const updated: WalletActionStepData = { ...stepData, version };
		const nextRaw = {
			...raw,
			with: walletActionStepConfig.serialize(updated)
		} as PipelineStepByType<'mobile-automation'>;

		return [nextRaw, updated as unknown as GenericRecord] as EnrichedStep;
	});
}
