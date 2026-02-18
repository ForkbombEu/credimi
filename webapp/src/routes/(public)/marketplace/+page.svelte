<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { baseSections, entities } from '$lib/global';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import { MarketplaceItemCard } from '$lib/marketplace';
	import ConformanceChecksTable from '$lib/marketplace/conformance-checks-table.svelte';
	import MarketplaceTable from '$lib/marketplace/marketplace-table.svelte';
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

	const sections = [...baseSections, entities.conformance_checks];
	const tabsParams = sections.map((t) => t.slug);

	const params = queryParameters({
		tab: {
			encode: (value) => value,
			decode: (value) => {
				if (value && tabsParams.includes(value)) return value;
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
		<div class="bg-secondary pt-10 pb-0">
			<div class="mx-auto max-w-7xl px-4 md:px-8">
				<T tag="h1" class="mb-8">
					{m.Marketplace()}
				</T>

				<div
					class="mb-8 flex flex-col gap-2 overflow-auto md:mb-0 md:flex-row md:items-stretch md:gap-0"
				>
					{#each sections as tab (tab.slug)}
						{@const isActive = params.tab === tab.slug}
						<button
							class={[
								'group rounded-md md:rounded-t-md md:rounded-b-none md:p-2',
								'flex items-stretch',
								{
									'bg-white text-primary': isActive,
									'shadow-md md:shadow-none': isActive
								}
							]}
							onclick={() => (params.tab = tab.slug)}
						>
							<div
								class={[
									'rounded-lg px-3 py-2 text-left leading-snug',
									'flex grow items-center gap-2',
									{
										'bg-primary/10 group-hover:bg-primary/20': !isActive
									}
								]}
							>
								<Icon src={tab.icon} class={[tab.classes.text, 'shrink-0']} />
								<div>
									{tab.labels.plural}
								</div>
							</div>
						</button>
					{/each}
				</div>

				{#if params.tab !== 'conformance-checks'}
					<div class="rounded-t-md bg-white px-4 pt-4 pb-6 md:rounded-t-none">
						<Search />
					</div>
				{/if}
			</div>
		</div>
	{/snippet}

	{#snippet contentWrapper(children)}
		<div class="min-h-[300px] grow bg-secondary">
			<div class="mx-auto max-w-7xl px-4 pb-8 md:px-8">
				{#if params.tab === 'conformance-checks'}
					<div class="rounded-lg rounded-tr-none bg-white pt-4">
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
			<div class="space-y-4">
				<PageGrid>
					{#each records as record (record.id)}
						<MarketplaceItemCard item={record} />
					{/each}
				</PageGrid>
				<Pagination />
			</div>
		{:else}
			<div in:fly={{ y: 10 }} class="space-y-4 rounded-b-md">
				<MarketplaceTable {records} />
				<Pagination />
			</div>
		{/if}
	{/snippet}
</CollectionManagerComponent>
