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
	import { userOrganization } from '$lib/app-state';
	import Button from '@/components/ui-custom/button.svelte';
	import { PencilIcon } from 'lucide-svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { CollectionForm } from '@/collections-components';
	import { pb } from '@/pocketbase';
	import Spinner from '@/components/ui-custom/spinner.svelte';

	//

	let { children, data } = $props();
	const { marketplaceItem } = $derived(data);

	const { logo, display } = $derived(getMarketplaceItemData(marketplaceItem));

	const isCurrentUserOwner = $derived(
		userOrganization.current?.id === marketplaceItem.organization_id
	);

	let isEditing = $state(false);
</script>

{#if isCurrentUserOwner}
	<div class="border-t-primary border-t-2 bg-[#E2DCF8] py-2">
		<div class="mx-auto flex max-w-screen-xl flex-wrap items-center gap-3 px-4 text-sm md:px-8">
			<T>{m.This_item_is_yours({ item: display.label })}</T>
			<Button size="sm" class="!h-8 text-xs" onclick={() => (isEditing = true)}>
				<PencilIcon />
				{m.Make_changes()}
			</Button>
			<T>{m.Last_edited()}: {new Date(marketplaceItem.updated).toLocaleDateString()}</T>
		</div>
	</div>
{/if}

<PageTop hideTopBorder={isCurrentUserOwner} contentClass="!space-y-4">
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

<Sheet bind:open={isEditing} hideTrigger>
	{#snippet content({ closeSheet })}
		<div class="flex flex-col gap-4">
			{#await pb.collection(marketplaceItem.type).getOne(marketplaceItem.id)}
				<Spinner />
			{:then record}
				<CollectionForm
					collection={marketplaceItem.type}
					recordId={marketplaceItem.id}
					initialData={record}
				/>
			{:catch error}
				<T>{error.message}</T>
			{/await}
		</div>
	{/snippet}
</Sheet>
