<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { Snippet } from 'svelte';

	import { userOrganization } from '$lib/app-state';
	import EntityTag from '$lib/global/entity-tag.svelte';

	import type { MarketplaceItemsResponse } from '@/pocketbase/types';

	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import type { MarketplaceItem } from './types';

	import TableNameCell from './_partials/table-name-cell.svelte';
	import { getMarketplaceItemData } from './utils';

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

	<TableNameCell {logo} name={typed.name} textToCopy={typed.path} {href}>
		{#if isCurrentUserOwner}
			<Badge class="block rounded-md">{m.Yours()}</Badge>
		{/if}
	</TableNameCell>
{/snippet}

{#snippet type(record: MarketplaceItemsResponse)}
	{@const { display } = getMarketplaceItemData(record as MarketplaceItem)}
	<EntityTag data={display} />
{/snippet}

{#snippet updated(record: MarketplaceItemsResponse)}
	<T class="text-muted-foreground">
		{new Date(record.updated as string).toLocaleDateString()}
	</T>
{/snippet}
