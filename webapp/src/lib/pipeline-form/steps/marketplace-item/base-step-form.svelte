<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T">
	import { getMarketplaceItemData } from '$lib/marketplace/utils.js';

	import { m } from '@/i18n/index.js';

	import type { StepType } from '../types';
	import type { BaseStepForm } from './base-step-form.svelte.js';

	import { getStepDisplayData } from '../_partials/display-data.js';
	import ItemCard from '../_partials/item-card.svelte';
	import SearchInput from '../_partials/search-input.svelte';
	import WithEmptyState from '../_partials/with-empty-state.svelte';
	import WithLabel from '../_partials/with-label.svelte';

	//

	type Props = {
		form: BaseStepForm<T>;
	};

	let { form }: Props = $props();

	const { label } = $derived(getStepDisplayData(form.collection as StepType));
</script>

<WithLabel {label} class="p-4">
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
