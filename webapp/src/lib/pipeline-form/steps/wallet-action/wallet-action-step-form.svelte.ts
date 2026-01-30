// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$lib/marketplace';

import { userOrganization } from '$lib/app-state/index.svelte.js';

import { pb } from '@/pocketbase/index.js';
import {
	Collections,
	type MobileRunnersResponse,
	type WalletActionsResponse,
	type WalletVersionsResponse
} from '@/pocketbase/types';

import { searchMarketplace } from '../_partials/search-marketplace';
import { Search } from '../_partials/search.svelte.js';
import { BaseForm } from '../types.js';
import Component from './wallet-action-step-form.svelte';

//

export interface WalletActionStepData {
	wallet: MarketplaceItem;
	version: WalletVersionsResponse;
	action: WalletActionsResponse;
	runner: MobileRunnersResponse;
}

export class WalletActionStepForm extends BaseForm<WalletActionStepData, WalletActionStepForm> {
	readonly Component = Component;

	constructor() {
		super();
		if (walletActionStepFormState.lastSelectedWallet) {
			this.data = {
				wallet: walletActionStepFormState.lastSelectedWallet.wallet,
				version: walletActionStepFormState.lastSelectedWallet.version,
				runner: walletActionStepFormState.lastSelectedWallet.runner,
				action: undefined
			};
		}
	}

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
			throw new Error('Invalid state');
		}
	});

	//

	foundWallets = $state<MarketplaceItem[]>([]);
	foundVersions = $state<WalletVersionsResponse[]>([]);
	foundRunners = $state<MobileRunnersResponse[]>([]);
	foundActions = $state<WalletActionsResponse[]>([]);

	walletSearch = new Search({
		onSearch: (text) => {
			this.searchWallet(text);
		}
	});

	async searchWallet(text: string) {
		this.foundWallets = await searchMarketplace(text, Collections.Wallets);
	}

	async selectWallet(wallet: MarketplaceItem) {
		this.data.wallet = wallet;
		this.foundVersions = await pb.collection('wallet_versions').getFullList({
			filter: `wallet = "${wallet.id}"`,
			requestKey: null
		});
	}

	selectVersion(version: WalletVersionsResponse) {
		this.data.version = version;
	}

	//

	runnerSearch = new Search({
		onSearch: (text) => {
			this.searchRunner(text);
		}
	});

	async searchRunner(text: string) {
		const filter = pb.filter(
			[
				['name ~ {:text}', 'canonified_name ~ {:text}'].join(' || '),
				['owner.id = {:currentOrganization}', 'published = true'].join(' || ')
			]
				.map((f) => `(${f})`)
				.join(' && '),
			{
				text: text,
				currentOrganization: userOrganization.current?.id
			}
		);
		this.foundRunners = await pb.collection('mobile_runners').getFullList({
			requestKey: null,
			filter: filter,
			sort: 'created'
		});
	}

	selectRunner(runner: MobileRunnersResponse) {
		this.data.runner = runner;
	}

	//

	actionSearch = new Search({
		onSearch: (text) => {
			this.searchAction(text);
		}
	});

	async searchAction(text: string) {
		const walletId = this.data.wallet?.id;
		if (!walletId) return;
		this.foundActions = await pb.collection('wallet_actions').getFullList({
			filter: `wallet = "${walletId}" && (name ~ "${text}" || canonified_name ~ "${text}")`,
			requestKey: null
		});
	}

	selectAction(action: WalletActionsResponse) {
		walletActionStepFormState.lastSelectedWallet = {
			wallet: this.data.wallet!,
			version: this.data.version!,
			runner: this.data.runner!
		};
		this.handleSubmit({ ...this.data, action } as WalletActionStepData);
	}

	//

	removeWallet() {
		this.data.wallet = undefined;
		this.data.version = undefined;
		this.data.runner = undefined;
		this.foundVersions = [];
		this.foundActions = [];
	}

	removeVersion() {
		this.data.version = undefined;
		this.data.runner = undefined;
	}

	removeRunner() {
		this.data.runner = undefined;
	}
}

//

type WalletActionStepFormState = {
	lastSelectedWallet:
		| {
				wallet: MarketplaceItem;
				version: WalletVersionsResponse;
				runner: MobileRunnersResponse;
		  }
		| undefined;
};

export const walletActionStepFormState = $state<WalletActionStepFormState>({
	lastSelectedWallet: undefined
});
