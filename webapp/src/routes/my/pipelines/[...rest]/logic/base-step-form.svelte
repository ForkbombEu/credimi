<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T">
	import { getMarketplaceItemData } from '$lib/marketplace/utils.js';

	import { m } from '@/i18n/index.js';

	import type { BaseStepForm } from './base-step-form.svelte.js';

	import ItemCard from './utils/item-card.svelte';
	import SearchInput from './utils/search-input.svelte';
	import WithEmptyState from './utils/with-empty-state.svelte';
	import WithLabel from './utils/with-label.svelte';

	//

	type Props = {
		form: BaseStepForm<T>;
	};

	let { form }: Props = $props();
</script>

<div class="space-y-4">
	<WithLabel label={m.Wallet()}>
		<SearchInput search={form.search} />
	</WithLabel>

	<WithEmptyState items={form.foundItems} emptyText={m.No_results_found()}>
		{#snippet item({ item })}
			<ItemCard
				avatar={getMarketplaceItemData(item).logo}
				title={item.name}
				subtitle={item.organization_name}
				onClick={() => form.selectItem(item)}
			/>
		{/snippet}
	</WithEmptyState>
</div>
