<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BackButton from '$lib/layout/back-button.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import { m } from '@/i18n';
	import { getMarketplaceItemData, MarketplaceItemTypeDisplay } from '../_utils';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';

	//

	let { children, data } = $props();
	const { marketplaceItem } = $derived(data);

	const { logo, display } = $derived(getMarketplaceItemData(marketplaceItem));
</script>

<PageTop contentClass="!space-y-4">
	<BackButton href="/marketplace">
		{m.Back_to_marketplace()}
	</BackButton>

	<div class="flex items-center gap-6">
		{#if logo}
			<Avatar src={logo} class="size-32 rounded-sm border" hideIfLoadingError />
		{/if}

		<div class="space-y-3">
			<div class="space-y-1">
				{#if display}
					<MarketplaceItemTypeDisplay data={display} />
				{/if}
				<T tag="h1">{marketplaceItem.name}</T>
			</div>
		</div>
	</div>
</PageTop>

<PageContent class="bg-secondary grow" contentClass="flex flex-col md:flex-row gap-12 items-start">
	{@render children()}
</PageContent>
