<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { Snippet } from 'svelte';

	import { userOrganization } from '$lib/app-state';
	import EntityTag from '$lib/global/entity-tag.svelte';
	import { getPath } from '$lib/utils';

	import type { HubItemsResponse } from '@/pocketbase/types';

	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import type { HubItem } from './types';

	import ContentWrapper from './_partials/content-wrapper.svelte';
	import TableNameCell from './_partials/table-name-cell.svelte';
	import { getHubItemData } from './utils';

	//

	const snippets = {
		name: name as Snippet<[HubItemsResponse]>,
		type: type as Snippet<[HubItemsResponse]>,
		updated: updated as Snippet<[HubItemsResponse]>,
		organization_name: organization_name as Snippet<[HubItemsResponse]>
	};
	export { snippets };
</script>

{#snippet name(record: HubItemsResponse)}
	{@const typed = record as HubItem}
	{@const { logo, href } = getHubItemData(typed)}
	{@const isCurrentUserOwner = userOrganization.current?.id === typed.organization_id}

	<TableNameCell {logo} name={typed.name} textToCopy={getPath(typed)} {href}>
		{#if isCurrentUserOwner}
			<Badge class="block rounded-md">{m.Yours()}</Badge>
		{/if}
	</TableNameCell>
{/snippet}

{#snippet type(record: HubItemsResponse)}
	{@const { display } = getHubItemData(record as HubItem)}
	<EntityTag data={display} />
{/snippet}

{#snippet updated(record: HubItemsResponse)}
	<ContentWrapper>
		<T class="text-muted-foreground">
			{new Date(record.updated as string).toLocaleDateString()}
		</T>
	</ContentWrapper>
{/snippet}

{#snippet organization_name(record: HubItemsResponse)}
	<ContentWrapper>
		<T>
			{record.organization_name}
		</T>
	</ContentWrapper>
{/snippet}
