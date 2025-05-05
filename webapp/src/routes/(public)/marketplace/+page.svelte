<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import CollectionManager from '@/collections-components/manager/collectionManager.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { truncate } from 'lodash';
	import {
		getMarketplaceItemData,
		getMarketplaceItemTypeData,
		type MarketplaceItem,
		MarketplaceItemTypeDisplay
	} from './_utils';
	import type { Filter } from '@/collections-components/manager';
	import Button from '@/components/ui-custom/button.svelte';

	const filters: Filter[] = [
		getMarketplaceItemTypeData('credential_issuers'),
		getMarketplaceItemTypeData('verifiers'),
		getMarketplaceItemTypeData('wallets')
	].map((item) => ({
		name: item.display?.label!,
		expression: item.filter
	}));
</script>

<CollectionManager
	collection="marketplace_items"
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
				<Filters>
					{#snippet trigger({ props })}
						<Button {...props} variant="outline" class="border-primary bg-secondary">
							{m.Filters()}
						</Button>
					{/snippet}
				</Filters>
			</div>
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
				{@const item = record as MarketplaceItem}
				{@const { href, logo, display } = getMarketplaceItemData(item)}

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
							<Avatar
								src={logo}
								class="size-14 !rounded-sm border"
								hideIfLoadingError
							/>
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
			{/each}
		</PageGrid>
	{/snippet}
</CollectionManager>
