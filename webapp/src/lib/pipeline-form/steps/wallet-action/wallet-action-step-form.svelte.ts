// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import { m } from '@/i18n/index.js';
import { type WalletActionsResponse, type WalletVersionsResponse } from '@/pocketbase/types';

import type { WalletActionStepData } from './types.js';

import { Search } from '../_partials/search.svelte.js';
import {
	EXTERNAL_VERSION,
	GLOBAL_RUNNER,
	type SelectedRunner,
	type SelectedVersion
} from '../../execution-target/types.js';
import { BaseForm, type InitFormOptions } from '../types.js';
import Component from './wallet-action-step-form.svelte';

//

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
		if (
			this.intent === 'add' &&
			this.isExecutionTargetLocked() &&
			wallet &&
			version &&
			runner &&
			!action
		) {
			return 'select-action';
		}
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
		} else {
			const target = this.getExecutionTarget();
			if (target) this.data = { ...target, action: undefined };
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
		this.defaultRunnerIfNeeded();
	}

	selectVersion(version: WalletVersionsResponse) {
		this.data.version = version;
		this.defaultRunnerIfNeeded();
	}

	selectExternalVersion() {
		this.data.version = EXTERNAL_VERSION;
		this.defaultRunnerIfNeeded();
	}

	private defaultRunnerIfNeeded() {
		if (this.isExecutionTargetLocked()) {
			return;
		}
		const target = this.getExecutionTarget();
		if (!target || target.runner === GLOBAL_RUNNER || target.runner === undefined) {
			this.data.runner = GLOBAL_RUNNER;
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
		this.data.action = action;
		this.commitIfAdding({ ...this.data, action } as WalletActionStepData);
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
