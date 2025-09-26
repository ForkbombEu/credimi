<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { MarketplaceItemsResponse } from '@/pocketbase/types';

	import { CollectionTable } from '@/collections-components/manager';
	import { m } from '@/i18n';

	import { snippets } from './marketplace-table-snippets.svelte';
	import TableRowAfter from './table-row-after.svelte';
	import {
		getIssuerItemCredentials,
		getMarketplaceItemTypeData,
		getVerifierItemUseCases,
		isCredentialIssuer,
		isVerifier
	} from './utils';

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
		<!-- 
		Note: instead of using an if to render conditionally
		We must render the TR no matter what
		in order for it to get the correct width
		And we hide it with a 'hidden' class
		  -->
		<TableRowAfter
			linksPromise={getIssuerItemCredentials(record).then((res) =>
				res.map((r) => ({
					title: r.display_name,
					href: `/marketplace/credentials/${r.id}`
				}))
			)}
			title={m.Credentials()}
			icon={getMarketplaceItemTypeData('credentials').display.icon}
			show={isCredentialIssuer(record)}
		/>
		<TableRowAfter
			linksPromise={getVerifierItemUseCases(record).then((res) =>
				res.map((r) => ({
					title: r.name,
					href: `/marketplace/use_cases_verifications/${r.id}`
				}))
			)}
			title={m.Verification_use_cases()}
			icon={getMarketplaceItemTypeData('use_cases_verifications').display.icon}
			show={isVerifier(record)}
		/>
	{/snippet}
</CollectionTable>
