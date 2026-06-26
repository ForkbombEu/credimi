// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import { isError } from 'effect/Predicate';

import type { EnrichedPipeline } from '../functions';
import type {
	SelectedRunner,
	SelectedVersion,
	WalletActionStepData
} from '../steps/wallet-action/wallet-action-step-form.svelte.js';

//

export interface Config {
	wallet: HubItem;
	version: SelectedVersion;
	runner: SelectedRunner;
}

const state = $state({
	current: undefined as Config | undefined
});

export function getCurrentConfig(): Config | undefined {
	return state.current;
}

/** Plain copy safe to use outside reactive context (e.g. form prefill, sync). */
export function getConfigClone(): Config | undefined {
	const current = state.current;
	if (!current) return undefined;
	return structuredClone(current);
}

export function setCurrentConfig(config: Config | undefined) {
	state.current = config === undefined ? undefined : structuredClone(config);
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

	const { wallet, version, runner } = data as unknown as WalletActionStepData;

	setCurrentConfig({ wallet, version, runner });
}

export function clear() {
	state.current = undefined;
}
