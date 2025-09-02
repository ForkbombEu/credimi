<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CardLink from '$lib/layout/card-link.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';

	import CollectionManager from '@/collections-components/manager/collectionManager.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
</script>

<CollectionManager queryOptions={{
	expand:[
		"custom_checks_via_owner", 
		"credentials_via_owner", 
		"credential_issuers_via_owner", 
		"verifiers_via_owner", 
		"wallets_via_owner"
	],
	filter: `
		custom_checks_via_owner.public = true || 
		credentials_via_owner.published = true || 
		credential_issuers_via_owner.published = true || 
		wallets_via_owner.published = true
	`,
	}} collection="organizations">
	{#snippet top({ Search })}
		<PageTop>
			<T tag="h1">{m.Find_providers_of_identity_solutions()}</T>
			<Search class="border-primary bg-secondary" />
		</PageTop>
	{/snippet}

	{#snippet contentWrapper(children)}
		<PageContent class="bg-secondary grow">
			{@render children()}
		</PageContent>
	{/snippet}

	{#snippet records({ records })}
		<PageGrid>
			{#each records as organization}
				{@const logoUrl = pb.files.getURL(organization, organization.logo)}
				<CardLink href={`/organizations/${organization.id}`} class="!p-4">
					<div class="flex items-center gap-4">
						<Avatar
							src={logoUrl}
							fallback={organization.name}
							class="!rounded-sm border size-12"
							hideIfLoadingError
						/>

						<T class="font-semibold">{organization.name}</T>
					</div>
				</CardLink>
			{/each}
		</PageGrid>
	{/snippet}
</CollectionManager>
