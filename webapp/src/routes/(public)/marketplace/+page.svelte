<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import { queryParameters } from 'sveltekit-search-params';

	import type { PocketbaseQueryOptions } from '@/pocketbase/query';

	import CollectionManager from '@/collections-components/manager/collectionManager.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import {
		getMarketplaceItemTypeData,
		MarketplaceItemCard,
		marketplaceItemTypes,
		marketplaceItemTypeSchema
	} from './_utils';

	//

	const params = queryParameters({
		type: {
			encode: (value) => value,
			decode: (value) => {
				try {
					return marketplaceItemTypeSchema.parse(value);
				} catch {
					return null;
				}
			}
		}
	});

	const typeFilter = $derived.by(() => {
		try {
			return marketplaceItemTypeSchema.parse(page.url.searchParams.get('type'));
		} catch {
			return undefined;
		}
	});

	const queryOptions: PocketbaseQueryOptions<'marketplace_items'> = $derived(
		typeFilter ? { filter: `type = '${typeFilter}'` } : {}
	);

	//

	// const filters: Filter[] = marketplaceItemTypes
	// 	.map((type) => getMarketplaceItemTypeData(type))
	// 	.map((item) => ({
	// 		name: item.display?.label!,
	// 		expression: item.filter
	// 	}));
</script>

<CollectionManager collection="marketplace_items" {queryOptions}>
	{#snippet top({ Search })}
		<PageTop>
			<div>
				<T tag="h1">
					<span> {m.Marketplace()}</span>
					{#if typeFilter}
						{@const typeData = getMarketplaceItemTypeData(typeFilter)}
						<span>/</span>
						<span class={typeData.display?.textClass}>
							{typeData.display?.labelPlural}
						</span>
					{/if}
				</T>
			</div>
			<div class="flex items-center gap-2">
				<Search
					containerClass="grow"
					class="border-primary bg-secondary                                                                                                                                                                                                                                                                                                              "
				/>
			</div>
		</PageTop>
	{/snippet}

	{#snippet contentWrapper(children)}
		<PageContent class="bg-secondary grow">
			<div class="flex flex-col gap-8 sm:flex-row">
				<div class="w-full sm:w-fit">
					{@render MarketplaceTableOfContents()}
				</div>
				<div class="grow">
					{@render children()}
				</div>
			</div>
		</PageContent>
	{/snippet}

	{#snippet records({ records })}
		<PageGrid>
			{#each records as record (record.id)}
				<MarketplaceItemCard item={record} />
			{/each}
		</PageGrid>
	{/snippet}
</CollectionManager>

{#snippet MarketplaceTableOfContents()}
	{@const isAllActive = params.type === null}
	<div class="grid grid-cols-2 sm:flex sm:flex-col">
		<Button
			variant={isAllActive ? 'default' : 'ghost'}
			size="sm"
			onclick={() => (params.type = null)}
			class="justify-start"
		>
			{m.All()}
		</Button>

		<div class="spacer relative sm:hidden"></div>

		{#each marketplaceItemTypes as type}
			{@const typeData = getMarketplaceItemTypeData(type)}
			{@const isActive = typeFilter === type}
			<Button
				variant={isActive ? 'default' : 'ghost'}
				size="sm"
				onclick={() => (params.type = type)}
				class={'justify-start '}
			>
				<div
					class={[
						'block size-3 shrink-0 rounded-full border border-white',
						typeData.display?.bgClass
					]}
				></div>
				{typeData.display?.labelPlural}
			</Button>
		{/each}
	</div>
{/snippet}
