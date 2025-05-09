<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CredentialCard from '$lib/layout/credentialCard.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import { CollectionManager } from '@/collections-components';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import type { FilterGroup } from '@/collections-components/manager';
	import { CredentialsFormatOptions } from '@/pocketbase/types';
	import { currentUser } from '@/pocketbase';

	const filters: FilterGroup[] = [
		{
			name: m.Format(),
			id: 'format',
			mode: '||',
			filters: Object.entries(CredentialsFormatOptions).map(([key, value]) => ({
				name: value,
				id: value,
				expression: `format='${value}'`
			}))
		}
	];
</script>

<CollectionManager
	collection="credentials"
	queryOptions={{
		searchFields: ['name', 'format'],
		perPage: 20,
		filter: `published = true`
	}}
	{filters}
>
	{#snippet top({ Search, Filters })}
		<PageTop>
			<T tag="h1">{m.Find_credential_attributes()}</T>
			<div class="flex items-center gap-2">
				<Search class="border-primary bg-secondary" containerClass="grow" />
				<Filters>
					{m.filters()}
				</Filters>
			</div>
		</PageTop>
	{/snippet}

	{#snippet contentWrapper(children)}
		<PageContent class="bg-secondary grow">
			{@render children()}
		</PageContent>
	{/snippet}

	{#snippet records({ records })}
		<PageGrid>
			{#each records as record (record.id)}
				<CredentialCard credential={record} />
			{/each}
		</PageGrid>
	{/snippet}
</CollectionManager>
