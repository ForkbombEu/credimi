// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';
import type { MarketplaceItem, MarketplaceItemType } from '$lib/marketplace';
import { Search } from '$lib/pipeline-form/steps/_partials/search.svelte';
import { BasePipelineStepDataForm, type PipelineStep } from '$lib/pipeline-form/types';
import { pb } from '@/pocketbase';
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
	onSelect: (item: MarketplaceItem) => PipelineStep;
	entityData: EntityData;
};

export class MarketplaceItemStepForm extends BasePipelineStepDataForm<MarketplaceItemStepForm> {
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
		this.handleSubmit(this.props.onSelect(item));
	}

	get collection() {
		return this.props.collection;
	}

	get entityData() {
		return this.props.entityData;
	}
}

//

async function searchMarketplace(path: string, type: MarketplaceItemType) {
	const result = await pb.collection('marketplace_items').getList(1, 10, {
		filter: pb.filter('path ~ {:path} && type = {:type}', { path, type }),
		requestKey: null
	});
	return result.items as MarketplaceItem[];
}
