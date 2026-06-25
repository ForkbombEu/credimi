// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import {
	type SelectedRunner,
	type SelectedVersion,
	type WalletActionStepData
} from '$pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.js';
import { isError } from 'effect/Predicate';

import type { EnrichedPipeline } from '../functions';
import {
	getSharedExecutionTargetContext,
	targetsEqual
} from '../steps-builder/_partials/shared-execution-target-context.js';
import type { EnrichedStep } from '../steps-builder/types.js';

//

export interface Config {
	wallet: HubItem;
	version: SelectedVersion;
	runner: SelectedRunner;
}

export const state = $state({
	current: undefined as Config | undefined,
	locked: false,
	secondStepPrefillSnapshot: undefined as Config | undefined
});

export function hasGlobalRunner() {
	return state.current?.runner === 'global';
}

export function hasUndefinedRunner() {
	return state.current?.runner === undefined;
}

export function syncFromSteps(steps: EnrichedStep[]) {
	const mobileSteps = steps.filter((step) => step[0].use === 'mobile-automation');

	if (mobileSteps.length === 0) {
		state.current = undefined;
		state.locked = false;
		return;
	}

	const lastStep = mobileSteps.at(-1)!;
	const [, data] = lastStep;
	if (isError(data)) {
		state.current = undefined;
		state.locked = false;
		return;
	}

	const { wallet, version, runner } = data as unknown as WalletActionStepData;

	state.current = {
		wallet,
		version,
		runner
	};
	state.locked = mobileSteps.length >= 2 && getSharedExecutionTargetContext(steps) !== null;
}

export function loadFromPipeline(pipeline: EnrichedPipeline) {
	syncFromSteps(pipeline.steps);
}

export function beginSecondStepAdd() {
	if (state.current) {
		state.secondStepPrefillSnapshot = { ...state.current };
	}
}

export function finishSecondStepAdd(submitted: Config) {
	if (
		state.secondStepPrefillSnapshot &&
		targetsEqual(state.secondStepPrefillSnapshot, submitted)
	) {
		state.locked = true;
	}
	state.secondStepPrefillSnapshot = undefined;
}

export function clear() {
	state.current = undefined;
	state.locked = false;
	state.secondStepPrefillSnapshot = undefined;
}

export function syncVersionIfSameWallet(walletId: string, version: SelectedVersion) {
	if (state.current?.wallet.id === walletId) {
		state.current = { ...state.current, version };
	}
}
