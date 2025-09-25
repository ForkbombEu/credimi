<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { MarketplaceItemsResponse } from '@/pocketbase/types';

	import { CollectionTable } from '@/collections-components/manager';

	import { snippets } from './marketplace-table-snippets.svelte';
	import TableIssuerRowAfter from './table-issuer-row-after.svelte';
	import { isCredentialIssuer, isVerifier } from './utils';

	type Props = {
		records: MarketplaceItemsResponse[];
	};

	let { records }: Props = $props();
</script>

<CollectionTable
	{records}
	hide={['delete', 'share', 'edit', 'select']}
	fields={['name', 'type', 'updated']}
	snippets={{
		name: snippets.name,
		type: snippets.type,
		updated: snippets.updated
	}}
	class="bg-background rounded-md"
	rowCellClass="px-4 py-2"
	headerClass="bg-background z-10"
>
	{#snippet rowAfter({ record })}
		{@const show = isCredentialIssuer(record) || isVerifier(record)}
		<!-- 
		Note: instead of using an if to render conditionally
		We must render the TR no matter what
		in order for it to get the correct width
		And we hide it with a 'hidden' class
		  -->
		<TableIssuerRowAfter issuer={record} {show} />
	{/snippet}
</CollectionTable>
