// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem, MarketplaceItemType } from '$lib/marketplace/utils.js';
import { StepFormState } from './types.js';
import { searchMarketplace } from './utils/search-marketplace.js';
import { Search } from './utils/search.svelte.js';

//

type Props<T> = {
	collection: MarketplaceItemType;
	onSelect: (item: MarketplaceItem) => T | Promise<T>;
};

export class BaseStepForm<T> extends StepFormState {
	constructor(private props: Props<T>) {
		super();
	}

	foundItems = $state<MarketplaceItem[]>([]);

	search = new Search({
		onSearch: (text) => {
			this.searchItem(text);
		}
	});

	async searchItem(text: string) {
		this.foundItems = await searchMarketplace(text, this.props.collection);
	}

	async selectItem(item: MarketplaceItem) {
		this.props.onSelect(item);
	}

	get collection() {
		return this.props.collection;
	}
}
