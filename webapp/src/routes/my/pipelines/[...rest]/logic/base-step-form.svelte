<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T">
	import type { ClassValue } from 'svelte/elements';

	import { getMarketplaceItemData } from '$lib/marketplace/utils.js';

	import { m } from '@/i18n/index.js';

	import type { BaseStepForm } from './base-step-form.svelte.js';
	import type { StepType } from './types.js';

	import { getStepDisplayData } from './utils/display-data.js';
	import ItemCard from './utils/item-card.svelte';
	import SearchInput from './utils/search-input.svelte';
	import WithEmptyState from './utils/with-empty-state.svelte';
	import WithLabel from './utils/with-label.svelte';

	//

	type Props = {
		form: BaseStepForm<T>;
		class?: ClassValue;
	};

	let { form, class: className }: Props = $props();

	const { label } = $derived(getStepDisplayData(form.collection as StepType));
</script>

<div class={['flex flex-col gap-4', className]}>
	<WithLabel {label}>
		<SearchInput search={form.search} />
	</WithLabel>

	<WithEmptyState items={form.foundItems} emptyText={m.No_results_found()} containerClass="grow">
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
