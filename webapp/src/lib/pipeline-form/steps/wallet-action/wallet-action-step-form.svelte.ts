// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import {
	EXTERNAL_VERSION,
	GLOBAL_DEVICE,
	type SelectedDevice,
	type SelectedVersion
} from '$pipeline-form/execution-target/types.js';
import { Search } from '$pipeline-form/steps/_partials/index.js';
import { BaseForm, type InitFormOptions } from '$pipeline-form/steps/types.js';

import { m } from '@/i18n/index.js';
import { type WalletActionsResponse, type WalletVersionsResponse } from '@/pocketbase/types';

import type { WalletActionStepData } from './types.js';

import Component from './wallet-action-step-form.svelte';

//

export function getVersionLabel(version: SelectedVersion) {
	return version === EXTERNAL_VERSION ? m.Installed_from_external_source() : `v. ${version.tag}`;
}

export function getDeviceLabel(device: SelectedDevice) {
	return device === GLOBAL_DEVICE ? m.Choose_later() : device.name;
}

//

export class WalletActionStepForm extends BaseForm<WalletActionStepData, WalletActionStepForm> {
	readonly Component = Component;

	data = $state<Partial<WalletActionStepData>>({});

	state = $derived.by(() => {
		const { wallet, version, action, device } = this.data;
		if (
			this.intent === 'add' &&
			this.isExecutionTargetLocked() &&
			wallet &&
			version &&
			device &&
			!action
		) {
			return 'select-action';
		}
		if (!wallet) {
			return 'select-wallet';
		} else if (wallet && !version) {
			return 'select-version';
		} else if (wallet && version && !device) {
			return 'select-device';
		} else if (wallet && version && device && !action) {
			return 'select-action';
		} else if (wallet && version && device && action) {
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
		this.defaultDeviceIfNeeded();
	}

	selectVersion(version: WalletVersionsResponse) {
		this.data.version = version;
		this.defaultDeviceIfNeeded();
	}

	selectExternalVersion() {
		this.data.version = EXTERNAL_VERSION;
		this.defaultDeviceIfNeeded();
	}

	private defaultDeviceIfNeeded() {
		if (this.isExecutionTargetLocked()) {
			return;
		}
		const target = this.getExecutionTarget();
		if (!target || target.device === GLOBAL_DEVICE || target.device === undefined) {
			this.data.device = GLOBAL_DEVICE;
		}
	}

	//

	deviceSearch = new Search({
		onSearch: () => {}
	});

	selectDevice(device: SelectedDevice) {
		this.data.device = device;
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
		this.data.device = undefined;
		this.data.action = undefined;
	}

	removeVersion() {
		this.data.version = undefined;
		this.data.device = undefined;
	}

	removeDevice() {
		this.data.device = undefined;
	}
}
