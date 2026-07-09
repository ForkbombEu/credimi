<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { HubItem } from '$lib/hub';
	import type { SelfProp } from '$lib/renderable';

	import { getHubItemLogo, getHubItemTypeFilter } from '$lib/hub/utils.js';
	import { ItemCard, StepCollectionPicker, WithLabel } from '$pipeline-form/steps/_partials/index.js';

	import type { HubItemStepForm } from './hub-item-step-form.svelte.js';

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

<StepCollectionPicker
	collection="hub_items"
	label={labels.singular}
	queryOptions={{
		filter: getHubItemTypeFilter(form.collection),
		searchFields: ['name']
	}}
	onSelect={(record) => form.selectItem(record as HubItem)}
>
	{#snippet item({ record, onSelect })}
		{@const item = record as HubItem}
		<ItemCard
			avatar={getHubItemLogo(item)}
			title={item.name}
			subtitle={item.organization_name}
			onClick={() => onSelect(record)}
		/>
	{/snippet}
</StepCollectionPicker>
