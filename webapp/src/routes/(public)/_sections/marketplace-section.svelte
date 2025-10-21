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
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import { MarketplaceItemCard } from '$lib/marketplace';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';

	let { findLabel, allLabel, collection }: SectionData = $props();

	const MAX_ITEMS = 3;
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<T tag="h3">{findLabel}</T>
		<Button variant="default" href="/marketplace?type={collection}">
			{allLabel}
		</Button>
	</div>

	<CollectionManager
		collection="marketplace_items"
		queryOptions={{ perPage: MAX_ITEMS, filter: `type = '${collection}'` }}
		hide={['pagination']}
	>
		{#snippet records({ records })}
			<PageGrid>
				{#each records as item, i (item.id)}
					{@const isLast = i == MAX_ITEMS - 1}
					<MarketplaceItemCard {item} class={isLast ? 'hidden lg:flex' : ''} />
				{/each}
			</PageGrid>
		{/snippet}
	</CollectionManager>
</div>
