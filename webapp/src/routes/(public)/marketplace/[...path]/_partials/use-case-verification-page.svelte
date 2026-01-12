<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { pb } from '@/pocketbase/index.js';
	import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';
	import { partitionPromises } from '@/utils/promise';

	import { pageDetails } from './_utils/types';

	export async function getUseCaseVerificationDetails(itemId: string, fetchFn = fetch) {
		const useCaseVerification = await new PocketbaseQueryAgent(
			{
				collection: 'use_cases_verifications',
				expand: ['verifier', 'credentials']
			},
			{ fetch: fetchFn }
		).getOne(itemId);

		// Handle potentially missing verifier
		const [verifierMarketplaceItems] = await partitionPromises([
			pb.collection('marketplace_items').getOne(useCaseVerification.verifier, { fetch: fetchFn })
		]);
		const verifierMarketplaceItem = verifierMarketplaceItems[0] ?? null;

		// Handle potentially missing credentials
		const [marketplaceCredentials] = await partitionPromises(
			useCaseVerification.credentials.map((c) =>
				pb.collection('marketplace_items').getOne(c, { fetch: fetchFn })
			)
		);

		return pageDetails('use_cases_verifications', {
			useCaseVerification,
			verifierMarketplaceItem,
			marketplaceCredentials
		});
	}
</script>

<script lang="ts">
	import NotFoundCard from '$lib/components/not-found-card.svelte';
	import MarketplaceItemCard from '$lib/marketplace/marketplace-item-card.svelte';

	import CodeSection from './_utils/code-section.svelte';
	import DescriptionSection from './_utils/description-section.svelte';
	import LayoutWithToc from './_utils/layout-with-toc.svelte';
	import PageSection from './_utils/page-section.svelte';
	import { sections as s } from './_utils/sections';
	import QrSection from './qr-section.svelte';

	//

	type Props = Awaited<ReturnType<typeof getUseCaseVerificationDetails>>;
	let { useCaseVerification, verifierMarketplaceItem, marketplaceCredentials }: Props = $props();
</script>

<LayoutWithToc
	sections={[
		s.description,
		s.qr_code,
		s.workflow_yaml,
		s.dcql_query,
		s.related_verifier,
		s.related_credentials
	]}
>
	<div class="flex items-start gap-6">
		<DescriptionSection description={useCaseVerification.description} class="grow" />
		<QrSection yaml={useCaseVerification.yaml} />
	</div>

	<CodeSection indexItem={s.workflow_yaml} code={useCaseVerification.yaml} language="yaml" />

	<CodeSection indexItem={s.dcql_query} code={useCaseVerification.dcql_query} language="json" />

	<div class="flex w-full flex-col gap-6 sm:flex-row">
		<PageSection indexItem={s.related_verifier} class="shrink-0 grow basis-1">
			{#if verifierMarketplaceItem}
				<MarketplaceItemCard item={verifierMarketplaceItem} />
			{:else}
				<NotFoundCard />
			{/if}
		</PageSection>

		<PageSection
			indexItem={s.related_credentials}
			empty={marketplaceCredentials.length === 0}
			class="shrink-0 grow basis-1"
		>
			<div class="flex flex-col gap-2">
				{#each marketplaceCredentials as marketplaceCredential (marketplaceCredential.id)}
					<MarketplaceItemCard item={marketplaceCredential} />
				{/each}
			</div>
		</PageSection>
	</div>
</LayoutWithToc>

<!-- 
<EditSheet>
	{#snippet children({ closeSheet })}
		<T tag="h2" class="mb-4">{m.Edit()} {useCaseVerification.name}</T>
		<CollectionForm
			collection="use_cases_verifications"
			recordId={useCaseVerification.id}
			initialData={useCaseVerification}
			{...options(useCaseVerification.owner, useCaseVerification.verifier)}
			onSuccess={closeSheet}
		/>
	{/snippet}
</EditSheet>
-->
