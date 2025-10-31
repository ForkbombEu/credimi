<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { Snippet } from 'svelte';

	import { userOrganization } from '$lib/app-state';

	import type { MarketplaceItemsResponse } from '@/pocketbase/types';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import { getMarketplaceItemData, MarketplaceItemTypeDisplay, type MarketplaceItem } from '.';

	//

	const snippets = {
		name: name as Snippet<[MarketplaceItemsResponse]>,
		type: type as Snippet<[MarketplaceItemsResponse]>,
		updated: updated as Snippet<[MarketplaceItemsResponse]>
	};
	export { snippets };
</script>

{#snippet name(record: MarketplaceItemsResponse)}
	{@const typed = record as MarketplaceItem}
	{@const { logo, href } = getMarketplaceItemData(typed)}
	{@const isCurrentUserOwner = userOrganization.current?.id === typed.organization_id}
	<div class="flex items-center gap-3">
		<Avatar
			src={logo ?? ''}
			class="size-10 !rounded-sm border"
			fallback={typed.name.slice(0, 2)}
		/>
		<div class="flex items-center gap-1">
			<a {href} class="hover:underline">
				<T class="overflow-hidden text-ellipsis font-semibold">{typed.name}</T>
			</a>
			<CopyButtonSmall textToCopy={typed.path} square variant="ghost" size="xs" />
		</div>
		{#if isCurrentUserOwner}
			<Badge class="block rounded-md">{m.Yours()}</Badge>
		{/if}
	</div>
{/snippet}

{#snippet type(record: MarketplaceItemsResponse)}
	{@const { display } = getMarketplaceItemData(record as MarketplaceItem)}
	<MarketplaceItemTypeDisplay data={display} />
{/snippet}

{#snippet updated(record: MarketplaceItemsResponse)}
	<T class="text-muted-foreground">
		{new Date(record.updated as string).toLocaleDateString()}
	</T>
{/snippet}
