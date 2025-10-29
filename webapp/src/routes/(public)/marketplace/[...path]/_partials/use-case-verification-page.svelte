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

		const verifierMarketplaceItem = await pb
			.collection('marketplace_items')
			.getOne(useCaseVerification.verifier, { fetch: fetchFn });

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
	import MarketplaceItemCard from '$lib/marketplace/marketplace-item-card.svelte';
	import { generateDeeplinkFromYaml } from '$lib/utils';
	import { onMount } from 'svelte';

	import QrStateful from '@/qr/qr-stateful.svelte';

	import CodeSection from './_utils/code-section.svelte';
	import DescriptionSection from './_utils/description-section.svelte';
	import LayoutWithToc from './_utils/layout-with-toc.svelte';
	import PageSection from './_utils/page-section.svelte';
	import { sections as s } from './_utils/sections';

	//

	type Props = Awaited<ReturnType<typeof getUseCaseVerificationDetails>>;
	let { useCaseVerification, verifierMarketplaceItem, marketplaceCredentials }: Props = $props();

	let qrLink = $state<string>('');
	let isProcessingYaml = $state(false);
	let yamlProcessingError = $state(false);

	//

	onMount(async () => {
		if (useCaseVerification.yaml) {
			isProcessingYaml = true;
			yamlProcessingError = false;
			try {
				const result = await generateDeeplinkFromYaml(useCaseVerification.yaml);
				if (result.deeplink) {
					qrLink = result.deeplink;
				}
			} catch (error) {
				console.error('Failed to process YAML for use case verification:', error);
				yamlProcessingError = true;
			} finally {
				isProcessingYaml = false;
			}
		}
	});

	//
</script>

<LayoutWithToc
	sections={[
		s.description,
		s.qr_code,
		s.workflow_yaml,
		s.related_verifier,
		s.related_credentials
	]}
>
	<div class="flex items-start gap-6">
		<DescriptionSection description={useCaseVerification.description} class="grow" />

		<PageSection indexItem={s.qr_code} class="space-y-0! flex flex-col items-stretch">
			<QrStateful
				src={qrLink}
				isLoading={isProcessingYaml}
				error={yamlProcessingError ? 'Dynamic generation failed' : undefined}
				loadingText="Processing YAML configuration..."
				placeholder="No credential offer available"
			/>
			<div class="w-60 break-all pt-4 text-xs">
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
				<a href={qrLink} target="_self">{qrLink}</a>
			</div>
		</PageSection>
	</div>

	<CodeSection indexItem={s.workflow_yaml} code={useCaseVerification.yaml} language="yaml" />

	<div class="flex w-full flex-col gap-6 sm:flex-row">
		<PageSection indexItem={s.related_verifier} class="shrink-0 grow basis-1">
			<MarketplaceItemCard item={verifierMarketplaceItem} />
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
