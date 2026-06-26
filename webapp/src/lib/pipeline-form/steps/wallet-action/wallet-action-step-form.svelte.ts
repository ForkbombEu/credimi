// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';
import type { Record } from '$lib/pipeline/runner';

import { ExecutionTarget } from '$lib/pipeline-form/execution-target';

import { m } from '@/i18n/index.js';
import { type WalletActionsResponse, type WalletVersionsResponse } from '@/pocketbase/types';

import { Search } from '../_partials/search.svelte.js';
import { BaseForm, type InitFormOptions } from '../types.js';
import Component from './wallet-action-step-form.svelte';

//

export const GLOBAL_RUNNER = 'global';
export const EXTERNAL_VERSION = 'installed_from_external_source';

export type SelectedRunner = Record | typeof GLOBAL_RUNNER;
export type SelectedVersion = WalletVersionsResponse | typeof EXTERNAL_VERSION;

export interface WalletActionStepData {
	wallet: HubItem;
	version: SelectedVersion;
	runner: SelectedRunner;
	action: WalletActionsResponse;
}

export function getVersionLabel(version: SelectedVersion) {
	return version === EXTERNAL_VERSION ? m.Installed_from_external_source() : `v. ${version.tag}`;
}

export function getRunnerLabel(runner: SelectedRunner) {
	return runner === GLOBAL_RUNNER ? m.Choose_later() : runner.name;
}

//

export class WalletActionStepForm extends BaseForm<WalletActionStepData, WalletActionStepForm> {
	readonly Component = Component;

	readonly lockExecutionTarget: boolean;

	data = $state<Partial<WalletActionStepData>>({});

	state = $derived.by(() => {
		const { wallet, version, action, runner } = this.data;
		if (!wallet) {
			return 'select-wallet';
		} else if (wallet && !version) {
			return 'select-version';
		} else if (wallet && version && !runner) {
			return 'select-runner';
		} else if (wallet && version && runner && !action) {
			return 'select-action';
		} else if (wallet && version && runner && action) {
			return 'ready';
		} else {
			throw new Error(m.Pipeline_form_invalid_state());
		}
	});

	constructor(opts?: InitFormOptions<WalletActionStepData>) {
		super(opts);
		this.lockExecutionTarget = opts?.lockExecutionTarget ?? false;
		if (opts?.initial) {
			this.data = { ...opts.initial };
		} else {
			const prefill = ExecutionTarget.getAddFormPrefill();
			if (prefill) this.data = { ...prefill, action: undefined };
		}
	}

	canSave() {
		return this.state === 'ready';
	}

	getSubmitData() {
		if (this.state !== 'ready') return undefined;
		return this.data as WalletActionStepData;
	}

	//

	selectWallet(wallet: HubItem) {
		const walletChanged = this.data.wallet?.id !== wallet.id;
		this.data.wallet = wallet;
		if (walletChanged) {
			this.clearWalletDependents();
		}
	}

	selectVersion(version: WalletVersionsResponse) {
		this.data.version = version;
		if (ExecutionTarget.shouldDefaultRunnerToGlobal()) {
			this.data.runner = 'global';
		}
	}

	selectExternalVersion() {
		this.data.version = EXTERNAL_VERSION;
		if (ExecutionTarget.shouldDefaultRunnerToGlobal()) {
			this.data.runner = 'global';
		}
	}

	//

	runnerSearch = new Search({
		onSearch: () => {}
	});

	selectRunner(runner: SelectedRunner) {
		this.data.runner = runner;
	}

	//

	selectAction(action: WalletActionsResponse) {
		ExecutionTarget.establishFromStep({
			wallet: this.data.wallet!,
			version: this.data.version!,
			runner: this.data.runner!,
			action
		} as WalletActionStepData);
		this.data.action = action;
		if (this.intent === 'add') {
			this.commit({ ...this.data, action } as WalletActionStepData);
		}
	}

	removeAction() {
		this.data.action = undefined;
	}

	//

	removeWallet() {
		this.data.wallet = undefined;
		this.clearWalletDependents();
	}

	removeVersion() {
		this.data.version = undefined;
		this.data.runner = undefined;
	}

	removeRunner() {
		this.data.runner = undefined;
	}

	private clearWalletDependents() {
		this.data.version = undefined;
		this.data.runner = undefined;
		this.data.action = undefined;
	}
}
