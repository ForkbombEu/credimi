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
		if (opts?.initial) {
			this.data = { ...opts.initial };
		} else if (ExecutionTarget.state.current) {
			this.data = {
				...ExecutionTarget.state.current,
				action: undefined
			};
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
		this.data.wallet = wallet;
		if (ExecutionTarget.hasGlobalRunner() || ExecutionTarget.hasUndefinedRunner()) {
			this.data.runner = 'global';
		}
	}

	selectVersion(version: WalletVersionsResponse) {
		this.data.version = version;
		if (ExecutionTarget.hasGlobalRunner() || ExecutionTarget.hasUndefinedRunner()) {
			this.data.runner = 'global';
		}
	}

	selectExternalVersion() {
		this.data.version = EXTERNAL_VERSION;
		if (ExecutionTarget.hasGlobalRunner() || ExecutionTarget.hasUndefinedRunner()) {
			this.data.runner = 'global';
		}
	}

	//

	runnerSearch = new Search({
		onSearch: () => {}
	});

	selectRunner(runner: ExecutionTarget.Config['runner']) {
		this.data.runner = runner;
	}

	//

	selectAction(action: WalletActionsResponse) {
		ExecutionTarget.state.current = {
			wallet: this.data.wallet!,
			version: this.data.version!,
			runner: this.data.runner!
		};
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
		this.data.version = undefined;
		this.data.runner = undefined;
	}

	removeVersion() {
		this.data.version = undefined;
		this.data.runner = undefined;
	}

	removeRunner() {
		this.data.runner = undefined;
	}
}
