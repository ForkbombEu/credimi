<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Array } from 'effect';

	import type { Link } from '@/components/types';
	import type { CollectionName } from '@/pocketbase/collections-models';
	import type { MarketplaceItemsResponse } from '@/pocketbase/types';

	import { CollectionTable } from '@/collections-components/manager';
	import { m } from '@/i18n';

	import type { MarketplaceItem } from './types';

	import TableChildrenCell from './_partials/table-children-cell.svelte';
	import { snippets } from './marketplace-table-snippets.svelte';
	import { isCredentialIssuer, isVerifier } from './utils';

	//

	type Props = {
		records: MarketplaceItemsResponse[];
	};

	let { records }: Props = $props();

	/* Children data */

	const collection = $derived.by(() => {
		const types = records.map((r) => r.type);
		const deduped = Array.dedupe(types);
		if (deduped.length === 1) return deduped[0] as CollectionName;
		else return undefined;
	});

	const childrenTitle = $derived.by(() => {
		switch (collection) {
			case 'credential_issuers':
				return m.Credentials();
			case 'verifiers':
				return m.Verification_use_cases();
			default:
				return false;
		}
	});

	function getChildrenLinks(record: MarketplaceItem): Link[] {
		if (!isCredentialIssuer(record) && !isVerifier(record)) return [];

		const type =
			record.type === 'credential_issuers' ? 'credentials' : 'use_cases_verifications';

		return (record.children ?? []).map((c) => ({
			title: c.name,
			href: `/marketplace/${type}/${record.organization_canonified_name}/${record.canonified_name}/${c.canonified_name}`
		}));
	}
</script>

<CollectionTable
	records={records as MarketplaceItem[]}
	hide={['delete', 'share', 'edit', 'select']}
	fields={['name', 'organization_name', 'updated']}
	snippets={{
		name: snippets.name,
		updated: snippets.updated,
		organization_name: snippets.organization_name
	}}
	class="rounded-md bg-background"
	rowCellClass="px-4 py-2 align-top"
	headerClass="bg-background z-10"
	labels={{
		organization_name: m.Organization(),
		updated: m.Last_update()
	}}
>
	{#snippet header({ Th })}
		{#if childrenTitle}
			<Th>
				{childrenTitle}
			</Th>
		{/if}
	{/snippet}

	{#snippet row({ record, Td })}
		{#if childrenTitle && record.children}
			{@const links = getChildrenLinks(record)}
			<Td>
				<TableChildrenCell items={links} />
			</Td>
		{/if}
	{/snippet}
</CollectionTable>
