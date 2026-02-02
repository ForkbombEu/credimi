// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$lib/marketplace';

import { isError, isString } from 'effect/Predicate';

import type { MobileRunnersResponse, WalletVersionsResponse } from '@/pocketbase/types';

import type { EnrichedPipeline } from '../functions';

//

export interface Config {
	wallet: MarketplaceItem;
	version: WalletVersionsResponse;
	runner: MobileRunnersResponse | 'global';
}

export const state = $state({
	current: undefined as Config | undefined
});

export function hasGlobalRunner() {
	return state.current?.runner === 'global';
}

export function hasUndefinedRunner() {
	return state.current?.runner === undefined;
}

export function loadFromPipeline(pipeline: EnrichedPipeline) {
	const steps = pipeline.steps.filter((step) => step[0].use === 'mobile-automation');

	const lastStep = steps.at(-1);
	if (!lastStep) {
		state.current = undefined;
		return;
	}

	const [, data] = lastStep;
	if (isError(data)) {
		state.current = undefined;
		return;
	}

	const { wallet, version, runner } = data as unknown as Config;

	const target: Config = {
		wallet: wallet,
		version: version,
		runner: 'global'
	};
	if (runner && !isString(runner)) {
		target.runner = runner;
	}

	state.current = target;
}

export function clear() {
	state.current = undefined;
}
