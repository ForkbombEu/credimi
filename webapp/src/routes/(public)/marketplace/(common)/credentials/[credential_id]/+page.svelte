<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import type { CredentialConfiguration } from '$lib/types/openid.js';
	import { QrCode } from '@/qr/index.js';
	import { String } from 'effect';
	import { MarketplaceItemCard, generateMarketplaceSection } from '../../../_utils/index.js';
	import MarketplacePageLayout from '$lib/layout/marketplace-page-layout.svelte';
	import { createIntentUrl } from '$lib/credentials/index.js';
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import { processYamlAndExtractCredentialOffer } from '$lib/compliance';
	import { onMount } from 'svelte';

	let { data } = $props();
	const { credential, credentialIssuer, credentialIssuerMarketplaceEntry } = $derived(data);

	// State for credential offer from YAML processing
	let credentialOfferFromYaml = $state<string | null>(null);
	let isProcessingYaml = $state(false);

	const sections = $derived(generateMarketplaceSection('credentials', {
		hasDescription: !!credential?.description,
		hasCompatibleIssuer: !!credentialIssuerMarketplaceEntry
	}));

	const credentialConfiguration = $derived(
		credential.json as CredentialConfiguration | undefined
	);

	// If credential has yaml and we got a credential offer from it, use that; otherwise use the default qr link
	const qrLink = $derived(
		credentialOfferFromYaml ||
		(String.isNonEmpty(credential.deeplink)
			? credential.deeplink
			: createIntentUrl(credential, credentialIssuer.url))
	);

	// Process YAML if it exists
	onMount(async () => {
		if (credential.yaml && String.isNonEmpty(credential.yaml)) {
			isProcessingYaml = true;
			try {
				const result = await processYamlAndExtractCredentialOffer(credential.yaml);
				if (result.success && result.credentialOffer) {
					credentialOfferFromYaml = result.credentialOffer;
				}
			} catch (error) {
				console.error('Failed to process YAML for credential offer:', error);
			} finally {
				isProcessingYaml = false;
			}
		}
	});
</script>

<MarketplacePageLayout tableOfContents={sections}>
	<div class="flex flex-col items-start gap-6 md:flex-row">
		<div class="grow space-y-6">
			<PageHeader
				title={sections.credential_properties.label}
				id={sections.credential_properties.anchor}
			/>

			<div class="flex gap-6">
				<InfoBox label="Issuer" value={credential.issuer_name} />
				<InfoBox label="Format" value={credential.format} />
				<InfoBox label="Locale" value={credential.locale} />
			</div>

			<div class="flex gap-6">
				<InfoBox
					label="Signing algorithms supported"
					value={credentialConfiguration?.credential_signing_alg_values_supported?.join(
						', '
					)}
				/>
				<InfoBox
					label="Cryptographic binding methods supported"
					value={credentialConfiguration?.cryptographic_binding_methods_supported?.join(
						', '
					)}
				/>
			</div>
			<InfoBox label="Type" value={credential.type} />
		</div>

		<div class="flex flex-col items-stretch">
			<PageHeader title="Credential offer" id="qr" />
			
			{#if isProcessingYaml}
				<div class="flex w-60 h-60 items-center justify-center border rounded-md">
					<div class="text-sm text-muted-foreground">Processing YAML configuration...</div>
				</div>
			{:else}
				<!-- Always use the same display format regardless of whether it's from YAML or default -->
				<QrCode src={qrLink} cellSize={10} class={['w-60 rounded-md']} />
				<div class="w-60 break-all pt-4 text-xs">
					<a href={qrLink} target="_self">{qrLink}</a>
				</div>
			{/if}
		</div>
	</div>

	{#if credential.description && sections.description}
		<div class="space-y-6">
			<PageHeader title={sections.description.label} id={sections.description.anchor} />
			<div class="prose">
				<RenderMd content={credential.description} />
			</div>
		</div>
	{/if}

	<div class="space-y-6">
		<PageHeader
			title={sections.credential_subjects.label}
			id={sections.credential_subjects.anchor}
		/>

		{#if credentialConfiguration}
			<CodeDisplay
				content={JSON.stringify(credentialConfiguration, null, 2)}
				language="json"
				class="border-primary bg-card text-card-foreground ring-primary w-fit max-w-screen-lg overflow-x-clip rounded-xl border p-6 text-xs shadow-sm transition-transform hover:-translate-y-2 hover:ring-2"
			/>
		{/if}
	</div>

	<div>
		<PageHeader
			title={sections.compatible_issuer.label}
			id={sections.compatible_issuer.anchor}
		/>

		{#if credentialIssuerMarketplaceEntry}
			<MarketplaceItemCard item={credentialIssuerMarketplaceEntry} />
		{/if}
	</div>
</MarketplacePageLayout>
