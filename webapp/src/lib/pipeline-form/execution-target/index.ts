// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import * as rules from './rules';
import * as state from './state.svelte';
import * as sync from './sync';

export const ExecutionTarget = {
	loadFromPipeline: state.loadFromPipeline,
	clear: state.clear,
	countMobileSteps: rules.countMobileSteps,
	shouldLockFormFields: rules.shouldLockFormFields,
	getAddFormPrefill: sync.getAddFormPrefill,
	getCurrentWallet: sync.getCurrentWallet,
	establishFromStep: sync.establishFromStep,
	shouldDefaultRunnerToGlobal: sync.shouldDefaultRunnerToGlobal,
	shouldOfferChooseRunnerLater: sync.shouldOfferChooseRunnerLater,
	syncAfterStepsChange: sync.syncAfterStepsChange,
	syncVersionIfSameWallet: sync.syncVersionIfSameWallet
};
