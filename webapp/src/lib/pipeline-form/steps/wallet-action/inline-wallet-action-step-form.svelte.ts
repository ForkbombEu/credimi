// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { BaseForm, type InitFormOptions } from '$pipeline-form/steps/types.js';

import type { InlineWalletActionStepData, WalletActionStepData } from './types.js';

import Component from './inline-wallet-action-step-form.svelte';
import { isInlineWalletActionStepData } from './types.js';

/**
 * Pass-through editor for inline action_code steps.
 * Preserves Maestro code on save; a full editor comes later.
 */
export class InlineWalletActionStepForm extends BaseForm<
	WalletActionStepData,
	InlineWalletActionStepForm
> {
	readonly Component = Component;

	data = $state<InlineWalletActionStepData | undefined>(undefined);

	constructor(opts?: InitFormOptions<WalletActionStepData>) {
		super(opts);
		if (opts?.initial && isInlineWalletActionStepData(opts.initial)) {
			this.data = { ...opts.initial };
		}
	}

	canSave() {
		return this.data !== undefined && this.data.actionCode.trim().length > 0;
	}

	getSubmitData(): WalletActionStepData | undefined {
		if (!this.canSave() || !this.data) return undefined;
		return this.data;
	}
}
