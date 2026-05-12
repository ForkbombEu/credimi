// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';
import type { MarketplaceItem, MarketplaceItemType } from '$lib/marketplace';
import { searchMarketplace } from '../_partials/search-marketplace';
import { Search } from '../_partials/search.svelte';
import { BaseForm } from '../types';
import Component from './marketplace-item-step-form.svelte';

//

const collections = [
	'credentials',
	'use_cases_verifications',
	'custom_checks'
] as const satisfies MarketplaceItemType[];

type MarketplaceStepCollection = (typeof collections)[number];

//

type Props = {
	collection: MarketplaceStepCollection;
	entityData: EntityData;
};

export class MarketplaceItemStepForm extends BaseForm<MarketplaceItem, MarketplaceItemStepForm> {
	readonly Component = Component;

	constructor(private props: Props) {
		super();
	}

	foundItems = $state<MarketplaceItem[]>([]);

	search = new Search({
		onSearch: (text) => {
			this.searchItem(text);
		}
	});

	async searchItem(text: string) {
		this.foundItems = await searchMarketplace(text, this.collection);
	}

	async selectItem(item: MarketplaceItem) {
		this.handleSubmit(item);
	}

	get collection() {
		return this.props.collection;
	}

	get entityData() {
		return this.props.entityData;
	}
}
