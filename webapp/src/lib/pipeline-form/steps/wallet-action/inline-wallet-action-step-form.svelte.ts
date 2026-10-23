// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { BaseForm, type InitFormOptions } from '$pipeline-form/steps/types.js';

import type { PassThroughWalletActionStepData, WalletActionStepData } from './types.js';

import Component from './inline-wallet-action-step-form.svelte';
import { isPassThroughWalletActionStepData } from './types.js';

/**
 * Pass-through editor for inline action_code steps and multi-action `actions`
 * steps. Preserves their actions on save; a full editor comes later.
 */
export class InlineWalletActionStepForm extends BaseForm<
	WalletActionStepData,
	InlineWalletActionStepForm
> {
	readonly Component = Component;

	data = $state<PassThroughWalletActionStepData | undefined>(undefined);

	constructor(opts?: InitFormOptions<WalletActionStepData>) {
		super(opts);
		if (opts?.initial && isPassThroughWalletActionStepData(opts.initial)) {
			this.data = { ...opts.initial };
		}
	}

	canSave() {
		if (this.data?.kind === 'actions') return this.data.actions.length > 0;
		return this.data !== undefined && this.data.actionCode.trim().length > 0;
	}

	getSubmitData(): WalletActionStepData | undefined {
		if (!this.canSave() || !this.data) return undefined;
		return this.data;
	}
}
