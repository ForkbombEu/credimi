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

	import { getMarketplaceItemTypeData, MarketplaceItemCard } from './_utils';
	import type { Filter } from '@/collections-components/manager';
	import Button from '@/components/ui-custom/button.svelte';

	//

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
				<MarketplaceItemCard item={record} />
			{/each}
		</PageGrid>
	{/snippet}
</CollectionManager>
