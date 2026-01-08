// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MarketplaceItem } from '$lib/marketplace';
import { StepFormState } from '$lib/pipeline-form/steps-builder/types';
import { Search } from '$lib/pipeline-form/steps/_partials/search.svelte';
import { searchMarketplace } from './utils/search-marketplace';

//

type Props<T> = {
	collection: MarketplaceStepType;
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
