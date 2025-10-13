<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

	import { X } from 'lucide-svelte';

	import type { MarketplaceItemsResponse } from '@/pocketbase/types';

	import { CollectionTable } from '@/collections-components/manager';
	import { m } from '@/i18n';

	import { snippets } from './marketplace-table-snippets.svelte';
	import TableRowAfter from './table-row-after.svelte';
	import {
		getMarketplaceItemTypeData,
		isCredentialIssuer,
		isVerifier,
		type MarketplaceItem
	} from './utils';

	type Props = {
		records: MarketplaceItemsResponse[];
	};

	let { records }: Props = $props();

	/* Row after data */

	/**
	  Note: We cannot use an if to render conditionally.
	  We must always render the TR in order for it to get the correct width.
	  Then, we hide it with a 'hidden' class
	*/

	type RowAfterProps = ComponentProps<typeof TableRowAfter>;

	// Dummy row data
	const dummyRowAfterProps: RowAfterProps = {
		items: [],
		title: '',
		icon: X,
		show: false
	};

	function getRowAfterProps(record: MarketplaceItemsResponse): RowAfterProps {
		if (!isCredentialIssuer(record) && !isVerifier(record)) return dummyRowAfterProps;
		const children = (record as MarketplaceItem).children ?? [];
		return {
			items: children.map((c) => ({
				title: c.name,
				href: `/marketplace/${record.type === 'credential_issuers' ? 'credentials' : 'use_cases_verifications'}/${record.organization_canonified_name}/${c.canonified_name}`
			})),
			title:
				record.type === 'credential_issuers' ? m.Credentials() : m.Verification_use_cases(),
			icon: getMarketplaceItemTypeData(record.type as MarketplaceItem['type']).display.icon,
			show: true
		};
	}
</script>

<CollectionTable
	{records}
	hide={['delete', 'share', 'edit', 'select']}
	fields={['name', 'organization_name', 'type', 'updated']}
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
		{@const props = getRowAfterProps(record)}
		<TableRowAfter {...props} />
	{/snippet}
</CollectionTable>
