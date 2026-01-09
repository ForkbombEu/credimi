<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { userOrganization } from '$lib/app-state';
	import EntityTag from '$lib/global/entity-tag.svelte';
	import { String } from 'effect';
	import { truncate } from 'lodash';
	import removeMd from 'remove-markdown';

	import type { MarketplaceItemsResponse } from '@/pocketbase/types';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import type { MarketplaceItem } from './types';

	import { getMarketplaceItemData } from './utils';

	//

	type Props = {
		item: MarketplaceItemsResponse;
		class?: string;
	};

	const { item: record, class: className = '' }: Props = $props();

	const item = $derived(record as MarketplaceItem);
	const { href, logo, display } = $derived(getMarketplaceItemData(item));

	const isCurrentUserOwner = $derived(userOrganization.current?.id === item.organization_id);

	const description = $derived.by(() => {
		const cleaned = removeMd(item.description ?? '');
		return truncate(cleaned, { length: 100 });
	});
</script>

<a
	{href}
	class={[
		'border-primary bg-card text-card-foreground ring-primary relative',
		'flex flex-col justify-between gap-4',
		'overflow-visible rounded-lg border p-4 shadow-sm transition-all hover:-translate-y-2 hover:ring-2',
		className
	]}
>
	<div class="flex items-start justify-between gap-4">
		<div>
			<p class="text-muted-foreground text-xs">{item.organization_name}</p>
			<T class="overflow-hidden text-ellipsis font-semibold">{item.name}</T>
		</div>

		<div class="flex flex-row-reverse flex-wrap items-start gap-1">
			{#if display}
				<EntityTag data={display} />
			{/if}
			{#if isCurrentUserOwner}
				<Badge class="block w-fit rounded-md py-[4px]">{m.Yours()}</Badge>
			{/if}
		</div>
	</div>

	{#if String.isNonEmpty(description)}
		<T class="text-muted-foreground pt-1 text-sm">
			{description}
		</T>
	{/if}

	<div class="flex items-end justify-between gap-2 pt-1">
		<T class="text-muted-foreground text-xs">
			{m.Last_update()}: {new Date(item.updated).toLocaleDateString()}
		</T>

		<Avatar
			src={logo ?? ''}
			class="size-12 !rounded-sm border"
			fallback={item.name.slice(0, 2)}
		/>
	</div>
</a>
