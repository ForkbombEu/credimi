<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CredentialCard from '$lib/layout/credentialCard.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { featureFlags } from '@/features';
	import { m } from '@/i18n';
	import { Collections, type CredentialsResponse } from '@/pocketbase/types';

	//

	const fakeCredential: CredentialsResponse = {
		id: 'das',
		created: '2024-12-12',
		updated: '2024-12-12',
		credential_issuer: 'das',
		json: {},
		key: 'das',
		description: 'Lorem ipsum',
		format: 'jwt_vc_json',
		issuer_name: 'das',
		logo: 'das',
		name: 'das',
		locale: 'en',
		type: 'plc',
		collectionId: '',
		collectionName: Collections.Credentials,
		deeplink: '',
		published: false,
		owner: 'das',
		conformant: false
	};
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<T tag="h3">{m.Find_solutions()}</T>
		<Button variant="default" href="/marketplace">{m.Explore_Marketplace()}</Button>
	</div>

	{#if $featureFlags.DEMO}
		<PageGrid class="select-none blur-sm">
			<CredentialCard credential={fakeCredential} class="pointer-events-none grow basis-1" />
			<CredentialCard credential={fakeCredential} class="pointer-events-none grow basis-1" />
			<CredentialCard
				credential={fakeCredential}
				class="pointer-events-none hidden grow basis-1 lg:block"
			/>
		</PageGrid>
	{:else}
		{@const MAX_ITEMS = 3}
		<CollectionManager
			collection="credentials"
			queryOptions={{
				perPage: MAX_ITEMS,
				filter: `published = true`
			}}
			hide={['pagination']}
		>
			{#snippet records({ records })}
				<PageGrid>
					{#each records as credential, i}
						{@const isLast = i == MAX_ITEMS - 1}
						<CredentialCard {credential} class={isLast ? 'hidden lg:flex' : ''} />
					{/each}
				</PageGrid>
			{/snippet}
		</CollectionManager>
	{/if}
</div>
