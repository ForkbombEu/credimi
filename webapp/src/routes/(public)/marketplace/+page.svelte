<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import { MarketplaceItemCard } from '$lib/marketplace';
	import ConformanceChecksTable from '$lib/marketplace/conformance-checks-table.svelte';
	import MarketplaceTable from '$lib/marketplace/marketplace-table.svelte';
	import { appSections } from '$lib/marketplace/sections';
	import { fly } from 'svelte/transition';
	import { queryParameters } from 'sveltekit-search-params';

	import type { PocketbaseQueryOptions } from '@/pocketbase/query';

	import CollectionManagerComponent from '@/collections-components/manager/collectionManager.svelte';
	import { CollectionManager } from '@/collections-components/manager/collectionManager.svelte.js';
	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	//

	let { data } = $props();

	const tabsParams = Object.values(appSections).map((t) => t.id);
	type TabParam = (typeof tabsParams)[number];

	const params = queryParameters({
		tab: {
			encode: (value) => value,
			decode: (value): TabParam => {
				if (value && tabsParams.includes(value as TabParam)) {
					return value as TabParam;
				}
				return 'wallets';
			}
		},
		mode: {
			encode: (value) => value,
			decode: (value) => (value === 'cards' ? 'cards' : 'table')
		}
	});

	let manager: CollectionManager<'marketplace_items'> | undefined;

	const queryOptions: PocketbaseQueryOptions<'marketplace_items'> = $derived.by(() => {
		switch (params.tab) {
			case 'wallets':
				return { filter: `type = 'wallets'` };
			case 'credential-issuers-and-credentials':
				return { filter: `type = 'credential_issuers' || type = 'credentials'` };
			case 'verifiers-and-use-case-verifications':
				return { filter: `type = 'verifiers' || type = 'use_cases_verifications'` };
			case 'custom-checks':
				return { filter: `type = 'custom_checks'` };
			case 'pipelines':
				return { filter: `type = 'pipelines'` };
			default:
				return {};
		}
	});

	$effect(() => {
		if (manager && params.tab) {
			manager.query.clearSearch();
		}
	});
</script>

<CollectionManagerComponent
	collection="marketplace_items"
	queryOptions={{ perPage: 25, searchFields: ['name'], ...queryOptions }}
	hide={['pagination']}
	onMount={(m) => {
		manager = m as CollectionManager<'marketplace_items'>;
	}}
>
	{#snippet top({ Search })}
		<div class="bg-secondary pb-10 pt-10 md:pb-0">
			<div class="mx-auto max-w-screen-xl px-4 md:px-8">
				<T tag="h1" class="mb-8">
					{m.Marketplace()}
				</T>

				<div class="flex flex-col gap-2 md:flex-row md:gap-0">
					{#each Object.values(appSections) as tab (tab.id)}
						{@const isActive = params.tab === tab.id}
						<button
							class={[
								'group rounded-md md:rounded-b-none md:rounded-t-md md:p-2',
								{
									'text-primary bg-white': isActive,
									'shadow-md md:shadow-none': isActive
								}
							]}
							onclick={() => (params.tab = tab.id)}
						>
							<div
								class={[
									'rounded-lg px-3 py-2 text-left',
									'flex items-center gap-2',
									{
										'group-hover:bg-primary/20 bg-primary/10': !isActive
									}
								]}
							>
								<Icon src={tab.icon} class={[tab.textClass, 'shrink-0']} />
								{tab.label}
							</div>
						</button>
					{/each}
				</div>

				{#if params.tab !== 'conformance-checks'}
					<div class="bg-white px-4 pb-6 pt-4">
						<Search />
					</div>
				{/if}
			</div>
		</div>
	{/snippet}

	{#snippet contentWrapper(children)}
		<div class="bg-secondary min-h-[300px] grow">
			<div class="mx-auto max-w-screen-xl px-4 pb-8 md:px-8">
				{#if params.tab === 'conformance-checks'}
					<div in:fly={{ y: 10 }} class="rounded-lg rounded-tr-none bg-white">
						<ConformanceChecksTable standardsWithTestSuites={data.conformanceChecks} />
					</div>
				{:else}
					{@render children()}
				{/if}
			</div>
		</div>
	{/snippet}

	{#snippet records({ records, Pagination })}
		{#if params.mode === 'cards' && params.tab !== 'conformance-checks'}
			<div in:fly={{ y: 10 }} class="space-y-4">
				<PageGrid>
					{#each records as record (record.id)}
						<MarketplaceItemCard item={record} />
					{/each}
				</PageGrid>
				<Pagination />
			</div>
		{:else}
			<div in:fly={{ y: 10 }} class="space-y-4">
				<MarketplaceTable {records} />
				<Pagination />
			</div>
		{/if}
	{/snippet}
</CollectionManagerComponent>
