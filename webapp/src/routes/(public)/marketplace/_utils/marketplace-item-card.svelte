<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { userOrganization } from '$lib/app-state';
	import { truncate } from 'lodash';

	import type { MarketplaceItemsResponse } from '@/pocketbase/types';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import { getMarketplaceItemData, type MarketplaceItem } from '.';
	import MarketplaceItemTypeDisplay from './marketplace-item-type-display.svelte';

	//

	type Props = {
		item: MarketplaceItemsResponse;
		class?: string;
	};

	const { item: record, class: className = '' }: Props = $props();

	const item = $derived(record as MarketplaceItem);
	const { href, logo, display } = $derived(getMarketplaceItemData(item));

	const isCurrentUserOwner = $derived(userOrganization.current?.id === item.organization_id);
</script>

<a
	{href}
	class="relative flex flex-col justify-between gap-2 overflow-hidden rounded-lg border border-primary bg-card p-6 text-card-foreground shadow-sm ring-primary transition-all hover:-translate-y-2 hover:ring-2 {className}"
>
	<div class="space-y-1">
		<T class="overflow-hidden text-ellipsis font-semibold">{item.name}</T>
		{#if display}
			<MarketplaceItemTypeDisplay data={display} />
		{/if}
		{#if item.description}
			<T class="pt-1 text-sm text-muted-foreground">
				{truncate(item.description, { length: 100 })}
			</T>
		{/if}
	</div>

	<div class="flex items-end justify-between gap-2">
		<T class="text-xs text-muted-foreground">
			{m.Last_update()}: {new Date(item.updated).toLocaleDateString()}
		</T>

		<Avatar
			src={logo ?? ''}
			class="size-12 !rounded-sm border"
			fallback={item.name.slice(0, 2)}
		/>
	</div>

	{#if isCurrentUserOwner}
		<div class="absolute right-0 top-0 p-1">
			<Badge class="block rounded-md">{m.Yours()}</Badge>
		</div>
	{/if}
</a>
