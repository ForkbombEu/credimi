<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { Collections } from '@/pocketbase/types';

	export type SectionData = {
		findLabel: string;
		allLabel: string;
		collection: Collections;
	};
</script>

<script lang="ts">
	import { MarketplaceItemCard } from '../marketplace/_utils';
	import T from '@/components/ui-custom/t.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import { CollectionManager } from '@/collections-components';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import { m } from '@/i18n';

	const MAX_ITEMS = 3;
    const excludeCollection = Collections.Credentials
     
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<T tag="h3">{m.Find_solutions()}</T>
		<Button variant="default" href="/marketplace">{m.Explore_Marketplace()}</Button>
	</div>

	<CollectionManager
		collection="marketplace_items"
		queryOptions={{ perPage: MAX_ITEMS, filter: `type != '${excludeCollection}'` }}
		hide={['pagination']}
	>
		{#snippet records({ records })}
			<PageGrid>
				{#each records as item, i}
					{@const isLast = i == MAX_ITEMS - 1}
					<MarketplaceItemCard {item} class={isLast ? 'hidden lg:flex' : ''} />
				{/each}
			</PageGrid>
		{/snippet}
	</CollectionManager>
</div>
