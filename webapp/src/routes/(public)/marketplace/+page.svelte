<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import { CornerDownRight } from 'lucide-svelte';
	import { nanoid } from 'nanoid';
	import { fly } from 'svelte/transition';
	import { queryParameters } from 'sveltekit-search-params';

	import type { PocketbaseQueryOptions } from '@/pocketbase/query';

	import { CollectionTable } from '@/collections-components/manager';
	import CollectionManager from '@/collections-components/manager/collectionManager.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { localizeHref, m } from '@/i18n';

	import {
		getIssuerItemCredentials,
		getMarketplaceItemTypeData,
		isCredentialIssuer,
		isVerifier,
		MarketplaceItemCard,
		marketplaceItemTypes,
		marketplaceItemTypeSchema
	} from './_utils';
	import { snippets } from './_utils/marketplace-table-snippets.svelte';

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
		},
		mode: {
			encode: (value) => value,
			decode: (value) => (value === 'table' ? 'table' : 'table')
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

<CollectionManager collection="marketplace_items" queryOptions={{ perPage: 25, ...queryOptions }}>
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
				<div class="w-full space-y-3 sm:w-fit">
					{@render MarketplaceTableOfContents()}
					<hr />
					{@render viewSwitcher()}
				</div>
				<div class="grow">
					{@render children()}
				</div>
			</div>
		</PageContent>
	{/snippet}

	{#snippet records({ records })}
		{#if params.mode === 'table'}
			<div in:fly={{ y: 10 }}>
				<CollectionTable
					{records}
					hide={['delete', 'share', 'edit', 'select']}
					fields={['name', 'type', 'updated']}
					snippets={{
						name: snippets.name,
						type: snippets.type,
						updated: snippets.updated
					}}
					class="bg-background rounded-md"
					rowCellClass="px-4 py-2"
					headerClass="bg-background z-10"
				>
					{#snippet rowAfter({ Tr, Td, record })}
						{@const show = isCredentialIssuer(record) || isVerifier(record)}
						{@const rowId = nanoid()}
						<Tr
							id={rowId}
							class={[
								'bg-gray-50 px-4',
								{ hidden: !show, 'hide-previous-border': show }
							]}
						>
							{#if isCredentialIssuer(record)}
								<Td class="py-1 text-xs" colspan={99}>
									<div class="flex w-full gap-4">
										{#await getIssuerItemCredentials(record) then credentials}
											<div class="flex items-center gap-1 text-gray-400">
												<CornerDownRight
													size={16}
													class="-translate-y-0.5"
												/>
												<span>{m.Credentials()} </span>
												<span class="w-6">
													({credentials.length})
												</span>
											</div>
											<div class="grid grid-cols-4">
												<!-- <div class="flex w-0 grow gap-1 overflow-x-scroll"> -->
												{#each credentials as credential}
													<a
														href={localizeHref(
															`/marketplace/credentials/${credential.id}`
														)}
														class="block truncate text-nowrap rounded-sm px-1 py-0.5 transition hover:bg-gray-300"
													>
														{credential.display_name}
													</a>
												{/each}
											</div>
										{/await}
									</div>
								</Td>
							{/if}
						</Tr>
					{/snippet}
				</CollectionTable>
			</div>
		{:else}
			<div in:fly={{ y: 10 }}>
				<PageGrid>
					{#each records as record (record.id)}
						<MarketplaceItemCard item={record} />
					{/each}
				</PageGrid>
			</div>
		{/if}
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
				class="justify-start"
			>
				{#if typeData.display?.icon}
					{@const IconComponent = typeData.display.icon}
					<IconComponent
						class="size-4 shrink-0 {isActive
							? 'text-primary-foreground'
							: `opacity-70 ${typeData.display?.textClass}`}"
					/>
				{/if}
				{typeData.display?.labelPlural}
			</Button>
		{/each}
	</div>
{/snippet}

{#snippet viewSwitcher()}
	<div class="px-3 text-sm">
		<span>
			{m.View()}:
		</span>
		{@render viewSwitcherLink('table')}
		<span>/</span>
		{@render viewSwitcherLink('card')}
	</div>
{/snippet}

{#snippet viewSwitcherLink(mode: 'table' | 'card')}
	<a
		href="/marketplace?mode={mode}"
		class={[
			'hover:underline',
			{
				'text-primary font-bold': params.mode === mode
			}
		]}
	>
		{#if mode === 'table'}
			{m.Table()}
		{:else}
			{m.Cards()}
		{/if}
	</a>
{/snippet}
