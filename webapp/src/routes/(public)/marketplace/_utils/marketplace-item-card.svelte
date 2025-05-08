<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { MarketplaceItemsResponse } from '@/pocketbase/types';
	import { getMarketplaceItemData, type MarketplaceItem } from '.';
	import T from '@/components/ui-custom/t.svelte';
	import { truncate } from 'lodash';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import MarketplaceItemTypeDisplay from './marketplace-item-type-display.svelte';

	type Props = {
		item: MarketplaceItemsResponse;
	};

	const { item: record }: Props = $props();

	const item = $derived(record as MarketplaceItem);
	const { href, logo, display } = $derived(getMarketplaceItemData(item));
</script>

<a
	{href}
	class="border-primary bg-card text-card-foreground ring-primary flex flex-col justify-between gap-4 rounded-lg border p-6 shadow-sm transition-all hover:-translate-y-2 hover:ring-2"
>
	<div class="space-y-1">
		<T class="overflow-hidden text-ellipsis font-semibold">{item.name}</T>
		{#if display}
			<MarketplaceItemTypeDisplay data={display} />
		{/if}
		{#if item.description}
			<T class="text-muted-foreground pt-1 text-sm">
				{truncate(item.description, { length: 100 })}
			</T>
		{/if}
	</div>

	<div class="flex items-end justify-between gap-2">
		<T class="text-muted-foreground text-xs">Last check: yyyy-mm-dd</T>
		{#if logo}
			<Avatar src={logo} class="size-14 !rounded-sm border" hideIfLoadingError />
		{/if}
	</div>

	<!-- <div
						class="text-muted-foreground flex flex-col items-start gap-2 overflow-hidden"
					>
						{#if String.isNonEmpty(service.url)}
							<T tag="small">{service.url}</T>
						{/if}
						{#if String.isNonEmpty(service.homepage_url)}
							<T tag="small">{service.homepage_url}</T>
						{/if}
						{#if String.isNonEmpty(service.repo_url)}
							<T tag="small">{service.repo_url}</T>
						{/if}
					</div> -->
</a>
