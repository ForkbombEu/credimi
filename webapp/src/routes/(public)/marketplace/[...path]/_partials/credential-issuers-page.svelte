<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { pb } from '@/pocketbase/index.js';
	import { PocketbaseQueryAgent } from '@/pocketbase/query';

	import { pageDetails } from './_utils/types';

	export async function getCredentialIssuersDetails(itemId: string, fetchFn = fetch) {
		const credentialIssuer = await new PocketbaseQueryAgent(
			{
				collection: 'credential_issuers',
				expand: ['credentials_via_credential_issuer']
			},
			{ fetch: fetchFn }
		).getOne(itemId);

		const credentialsIds = (
			credentialIssuer.expand?.credentials_via_credential_issuer ?? []
		).map((credential) => credential.id);

		const credentialsFilters = credentialsIds.map((id) => `id = '${id}'`).join(' || ');

		const credentialsMarketplaceItems =
			credentialsFilters.length > 0
				? await pb.collection('marketplace_items').getFullList(1, {
						filter: credentialsFilters,
						fetch: fetchFn
					})
				: [];

		return pageDetails('credential_issuers', {
			credentialIssuer,
			credentialsMarketplaceItems
		});
	}
</script>

<script lang="ts">
	import EmptyState from '$lib/layout/empty-state.svelte';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import PageHeaderIndexed from '$lib/layout/pageHeaderIndexed.svelte';
	import { MarketplaceItemCard } from '$marketplace/_utils';
	import { String } from 'effect';

	import { CollectionForm } from '@/collections-components/index.js';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import EditSheet from './_utils/edit-sheet.svelte';
	import LayoutWithToc from './_utils/layout-with-toc.svelte';
	import { sections as s } from './_utils/sections';

	//

	type Props = Awaited<ReturnType<typeof getCredentialIssuersDetails>>;

	let { credentialIssuer, credentialsMarketplaceItems }: Props = $props();
</script>

<LayoutWithToc sections={[s.general_info, s.description, s.credentials]}>
	<div class="space-y-6">
		<PageHeaderIndexed indexItem={s.general_info} />

		<InfoBox label="URL" url={credentialIssuer.url} copyable={true} />

		{#if String.isNonEmpty(credentialIssuer.repo_url)}
			<InfoBox label="Repository" url={credentialIssuer.repo_url} copyable={true} />
		{/if}

		{#if String.isNonEmpty(credentialIssuer.homepage_url)}
			<InfoBox label="Homepage" url={credentialIssuer.homepage_url} copyable={true} />
		{/if}
	</div>

	<div class="space-y-6">
		<PageHeaderIndexed indexItem={s.description} />
		{#if credentialIssuer.description}
			<div class="prose">
				<RenderMd content={credentialIssuer.description} />
			</div>
		{:else}
			<EmptyState>
				<T>{m.No_information_available()}</T>
			</EmptyState>
		{/if}
	</div>

	<div class="space-y-6">
		<PageHeaderIndexed indexItem={s.credentials} />

		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			{#each credentialsMarketplaceItems as credential (credential.id)}
				<MarketplaceItemCard item={credential} />
			{:else}
				<EmptyState>
					<T>No credentials found</T>
				</EmptyState>
			{/each}
		</div>
	</div>
</LayoutWithToc>

<EditSheet>
	{#snippet children({ closeSheet })}
		<T tag="h2" class="mb-4">{m.Edit()} {credentialIssuer.name}</T>
		<CollectionForm
			collection="credential_issuers"
			recordId={credentialIssuer.id}
			initialData={credentialIssuer}
			onSuccess={closeSheet}
			fieldsOptions={{
				exclude: ['owner', 'url', 'published', 'imported', 'workflow_url']
			}}
		/>
	{/snippet}
</EditSheet>
