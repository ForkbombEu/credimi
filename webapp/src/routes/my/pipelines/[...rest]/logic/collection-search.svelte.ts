// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CollectionName } from '@/pocketbase/collections-models';
import { PocketbaseQueryAgent, type PocketbaseQueryExpandOption } from '@/pocketbase/query';
import type { CollectionResponses } from '@/pocketbase/types';
import type { KeyOf } from '@/utils/types';
import { Debounced } from 'runed';

export class CollectionSearch<
	C extends CollectionName,
	E extends PocketbaseQueryExpandOption<C> = never
> {
	searchText = $state('');
	private debouncedSearchText = new Debounced(() => this.searchText, 500);

	constructor(
		public readonly collection: C,
		public readonly expand: E,
		public readonly searchFields: KeyOf<CollectionResponses[C]>[]
	) {
		$effect(() => {
			this.debouncedSearchText;
		});
	}

	private performSearch(text: string) {
		const runners = new PocketbaseQueryAgent({
			collection: this.collection,
			expand: this.expand,
			search: text,
			searchFields: this.searchFields
		});
		runners.getFullList().then((res) => {
			this.results = res;
		});
	}
}
