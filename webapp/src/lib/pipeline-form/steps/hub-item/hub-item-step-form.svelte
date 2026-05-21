<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { getHubItemLogo } from '$lib/hub/utils.js';

	import { m } from '@/i18n/index.js';

	import type { HubItemStepForm } from './hub-item-step-form.svelte.js';

	import ItemCard from '../_partials/item-card.svelte';
	import SearchInput from '../_partials/search-input.svelte';
	import WithEmptyState from '../_partials/with-empty-state.svelte';
	import WithLabel from '../_partials/with-label.svelte';

	//

	let { self: form }: SelfProp<HubItemStepForm> = $props();

	const { labels } = $derived(form.entityData);
</script>

{#if form.selectedItem}
	<div class="border-b p-4">
		<WithLabel label={labels.singular}>
			<ItemCard
				avatar={getHubItemLogo(form.selectedItem)}
				title={form.selectedItem.name}
				subtitle={form.selectedItem.organization_name}
				onDiscard={() => form.discardSelection()}
			/>
		</WithLabel>
	</div>
{/if}

<WithLabel label={labels.singular} class="p-4">
	<SearchInput search={form.search} />
</WithLabel>

<WithEmptyState items={form.foundItems} emptyText={m.No_results_found()}>
	{#snippet item({ item })}
		<ItemCard
			avatar={getHubItemLogo(item)}
			title={item.name}
			subtitle={item.organization_name}
			onClick={() => form.selectItem(item)}
		/>
	{/snippet}
</WithEmptyState>
