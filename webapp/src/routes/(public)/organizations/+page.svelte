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

<CollectionManager collection="organization_info">
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
				<CardLink href={`/organizations/${organization.id}`}>
					<div class="flex items-center gap-4">
						{#if organization.logo}
							{@const logoUrl = pb.files.getURL(organization, organization.logo)}
							<Avatar src={logoUrl} class="!rounded-sm" hideIfLoadingError />
						{/if}
						<T class="font-semibold">{organization.name}</T>
					</div>
				</CardLink>
			{/each}
		</PageGrid>
	{/snippet}
</CollectionManager>
