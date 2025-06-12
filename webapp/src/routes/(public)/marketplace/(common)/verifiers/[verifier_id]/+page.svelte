<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { Building2, Key, Layers, ScanEye } from 'lucide-svelte';
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import { String } from 'effect';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import MarketplacePageLayout from '$lib/layout/marketplace-page-layout.svelte';
	import { MarketplaceItemCard } from '$marketplace/_utils/index.js';

	//

	let { data } = $props();
	const { verifier, marketplaceCredentials, marketplaceVerificationUseCases } = $derived(data);

	//

	const sections = {
		general_info: {
			icon: Building2,
			anchor: 'general_info',
			label: m.General_info()
		},
		description: {
			icon: Layers,
			anchor: 'description',
			label: m.Description()
		},
		credentials: {
			icon: Key,
			anchor: 'credentials',
			label: m.Linked_credentials()
		},
		verification_use_cases: {
			icon: ScanEye,
			anchor: 'verification_use_cases',
			label: m.Verification_use_cases()
		}
	} satisfies Record<string, IndexItem>;

	//

	const standardAndVersion = $derived(verifier.standard_and_version.split(','));
</script>

<MarketplacePageLayout tableOfContents={sections}>
	<div class="space-y-6">
		<PageHeader title={sections.general_info.label} id={sections.general_info.anchor} />

		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			<InfoBox label="URL">
				<a href={verifier.url} class="hover:underline" target="_blank">
					{verifier.url}
				</a>
			</InfoBox>

			{#if String.isNonEmpty(verifier.repository_url)}
				<InfoBox label="Homepage">
					<a href={verifier.repository_url} class="hover:underline" target="_blank">
						{verifier.repository_url}
					</a>
				</InfoBox>
			{:else}
				<div></div>
			{/if}

			<InfoBox label={m.Standard_and_version()}>
				<ul class="">
					{#each standardAndVersion as standard}
						<li>{standard}</li>
					{/each}
				</ul>
			</InfoBox>

			<InfoBox label={m.Signing_algorithms_supported()}>
				<T>{verifier.signing_algorithms.join(', ')}</T>
			</InfoBox>

			<InfoBox label={m.Cryptographic_binding_methods_supported()}>
				<T>{verifier.cryptographic_binding_methods.join(', ')}</T>
			</InfoBox>

			<InfoBox label={m.Credentials_format()}>
				<T>{verifier.format.join(', ')}</T>
			</InfoBox>
		</div>
	</div>

	<div class="space-y-6">
		<PageHeader title={sections.description.label} id={sections.description.anchor} />

		<div class="prose">
			<RenderMd content={verifier.description} />
		</div>
	</div>

	<div class="space-y-6">
		<PageHeader title={sections.credentials.label} id={sections.credentials.anchor} />

		{#if marketplaceCredentials.length > 0}
			<PageGrid>
				{#each marketplaceCredentials as credential}
					<MarketplaceItemCard item={credential} />
				{/each}
			</PageGrid>
		{:else}
			<T>{m.No_published_credentials_found()}</T>
		{/if}
	</div>

	<div class="space-y-6">
		<PageHeader
			title={sections.verification_use_cases.label}
			id={sections.verification_use_cases.anchor}
		/>

		{#if marketplaceVerificationUseCases.length > 0}
			<PageGrid>
				{#each marketplaceVerificationUseCases as verificationUseCase}
					<MarketplaceItemCard item={verificationUseCase} />
				{/each}
			</PageGrid>
		{:else}
			<T>{m.No_published_verification_use_cases_found()}</T>
		{/if}
	</div>
</MarketplacePageLayout>
