<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { getMarketplaceItemLogo } from '$lib/marketplace/utils.js';

	import { m } from '@/i18n/index.js';

	import type { MarketplaceItemStepForm } from './marketplace-item-step-form.svelte.js';

	import ItemCard from '../_partials/item-card.svelte';
	import SearchInput from '../_partials/search-input.svelte';
	import WithEmptyState from '../_partials/with-empty-state.svelte';
	import WithLabel from '../_partials/with-label.svelte';

	//

	let { self: form }: SelfProp<MarketplaceItemStepForm> = $props();

	const { labels } = $derived(form.entityData);
</script>

<WithLabel label={labels.singular} class="p-4">
	<SearchInput search={form.search} />
</WithLabel>

<WithEmptyState items={form.foundItems} emptyText={m.No_results_found()}>
	{#snippet item({ item })}
		<ItemCard
			avatar={getMarketplaceItemLogo(item)}
			title={item.name}
			subtitle={item.organization_name}
			onClick={() => form.selectItem(item)}
		/>
	{/snippet}
</WithEmptyState>
