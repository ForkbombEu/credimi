// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$lib/marketplace';
import { pb } from '@/pocketbase/index.js';
import {
	Collections,
	type WalletActionsResponse,
	type WalletVersionsResponse
} from '@/pocketbase/types';
import { StepFormState, type WalletStepData } from '../types.js';
import { searchMarketplace } from '../utils/search-marketplace.js';
import { Search } from '../utils/search.svelte.js';

//

type Props = {
	onSelect: (step: WalletStepData) => void;
	initialData?: Partial<WalletStepData>;
};

export class WalletStepForm extends StepFormState {
	constructor(private props: Props) {
		super();
		if (props.initialData) this.data = props.initialData;
	}

	data = $state<Partial<WalletStepData>>({});

	state = $derived.by(() => {
		const { wallet, version, action } = this.data;
		if (!wallet) {
			return 'select-wallet';
		} else if (wallet && !version) {
			return 'select-version';
		} else if (wallet && version && !action) {
			return 'select-action';
		} else if (wallet && version && action) {
			return 'ready';
		} else {
			throw new Error('Invalid state');
		}
	});

	//

	foundWallets = $state<MarketplaceItem[]>([]);
	foundVersions = $state<WalletVersionsResponse[]>([]);
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
		this.props.onSelect({ ...this.data, action } as WalletStepData);
	}

	//

	removeWallet() {
		console.log('removeWallet');
		this.data.wallet = undefined;
		this.foundVersions = [];
		this.foundActions = [];
	}

	removeVersion() {
		this.data.version = undefined;
	}
}
