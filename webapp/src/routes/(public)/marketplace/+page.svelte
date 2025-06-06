<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import CollectionManager from '@/collections-components/manager/collectionManager.svelte';

	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import {
		getMarketplaceItemTypeData,
		MarketplaceItemCard,
		marketplaceItemTypes,
		marketplaceItemTypeSchema,
		type MarketplaceItemType
	} from './_utils';
	import type { Filter } from '@/collections-components/manager';
	import Button from '@/components/ui-custom/button.svelte';
	import { page } from '$app/state';
	import type { PocketbaseQueryOptions } from '@/pocketbase/query';
	import type { MaybeArray } from '@/utils/other';
	import {type MarketplaceItemsRecord } from '@/pocketbase/types';
	import type { SortOption } from '@/pocketbase/query/query';

	//

	const type = $derived.by(() => {
		try {
			return marketplaceItemTypeSchema.parse(page.url.searchParams.get('type'));
		} catch (error) {
			return undefined;
		}
	});

	const fromDate = $derived.by(() => {
		const date = page.url.searchParams.get('from');
		return date ? date : undefined;
	});

	const tillDate = $derived.by(() => {
		const date = page.url.searchParams.get('to');
		console.log('tillDate', decodeURI(date || ''));
		return date ? date : undefined;
	});

	type SortType = MaybeArray<SortOption<"expand" | keyof MarketplaceItemsRecord>>
	
	const orderBy = $derived.by(() => {
		const orderBy = page.url.searchParams.get('order-by');
		let order = page.url.searchParams.get('order');
		if (order == null || !(order == 'DESC' || order == 'ASC')) order = 'ASC';
		try {
			if (orderBy == null) return undefined;
			console.log('orderBy', orderBy, 'order', order);
			return [orderBy, order] as SortType;
		} catch (error) {
			console.error('Invalid orderBy parameter:', error);
		}
		return undefined;
	});

	const name = $derived.by(() => {
		return page.url.searchParams.get('name');
	});

	const perPage = $derived.by(() => {
		const perPageParam = page.url.searchParams.get('perPage');
		return perPageParam ? parseInt(perPageParam, 10) : 20;
	});

	const queryOptions: PocketbaseQueryOptions<'marketplace_items'> = $derived.by(() => {
		const filterParts: string[] = [];
		if (type) {
			filterParts.push(`type = '${type}'`);
		}
		if (fromDate) {
			filterParts.push(`updated >= '${fromDate}'`);
		}
		if (tillDate) {
			filterParts.push(`updated <= '${tillDate}'`);
		}
		return {
			filter: filterParts.join(' && '),
			sort: orderBy,
			search: name,
			perPage: perPage || 20
		};
	});
	//

	const filters: Filter[] = marketplaceItemTypes
		.map((type) => getMarketplaceItemTypeData(type))
		.map((item) => ({
			name: item.display?.label!,
			expression: item.filter
		}));
</script>

<CollectionManager
	collection="marketplace_items"
	{queryOptions}
	filters={{
		name: m.Type(),
		id: 'default',
		mode: '||',
		filters: filters
	}}
>
	{#snippet top({ Search, Filters })}
		<PageTop>
			<T tag="h1">{m.Marketplace()}</T>
			<div class="flex items-center gap-2">
				<Search
					containerClass="grow"
					class="border-primary bg-secondary                                                                                                                                                                                                                                                                                                              "
				/>
				{#if !type}
					<Filters>
						{#snippet trigger({ props })}
							<Button
								{...props}
								variant="outline"
								class="border-primary bg-secondary"
							>
								{m.Filters()}
							</Button>
						{/snippet}
					</Filters>
				{/if}
			</div>
			{#if type}
				{@const typeData = getMarketplaceItemTypeData(type)}
				<div class="flex items-center gap-2">
					<T>
						{m.Filters()}:
						<span class={typeData.display?.textClass}>{typeData.display?.label}</span>
					</T>
					<Button variant="outline" href="/marketplace" size="sm">{m.Reset()}</Button>
				</div>
			{/if}
		</PageTop>
	{/snippet}

	{#snippet contentWrapper(children)}
		<PageContent class="bg-secondary grow">
			{@render children()}
		</PageContent>
	{/snippet}

	{#snippet records({ records })}
		<PageGrid>
			{#each records as record}
				<MarketplaceItemCard item={record} />
			{/each}
		</PageGrid>
	{/snippet}
</CollectionManager>
