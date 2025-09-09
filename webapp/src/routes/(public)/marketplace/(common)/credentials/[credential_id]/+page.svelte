<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { CredentialConfiguration } from '$lib/types/openid.js';

	import { createIntentUrl } from '$lib/credentials/index.js';
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import MarketplacePageLayout from '$lib/layout/marketplace-page-layout.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import EditCredentialForm from '$routes/my/services-and-products/_credentials/credential-form.svelte';
	import { String } from 'effect';
	import { TriangleAlert } from 'lucide-svelte';
	import { onMount } from 'svelte';
	import { z } from 'zod';

	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import { QrCode } from '@/qr/index.js';

	import EditSheet from '../../_utils/edit-sheet.svelte';
	import { MarketplaceItemCard, generateMarketplaceSection } from '../../../_utils/index.js';

	//

	let { data } = $props();
	const { credential, credentialIssuer, credentialIssuerMarketplaceEntry } = $derived(data);

	// State for credential offer from YAML processing
	let credentialOfferFromYaml = $state<string | null>(null);
	let isProcessingYaml = $state(false);
	let yamlProcessingError = $state(false);

	const sections = $derived(
		generateMarketplaceSection('credentials', {
			hasDescription: !!credential?.description,
			hasCompatibleIssuer: !!credentialIssuerMarketplaceEntry
		})
	);

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
			yamlProcessingError = false;
			try {
				const result = await processYamlAndExtractCredentialOffer(credential.yaml);
				if (result.credentialOffer) {
					credentialOfferFromYaml = result.credentialOffer;
				}
			} catch (error) {
				console.error('Failed to process YAML for credential offer:', error);
				yamlProcessingError = true;
				credentialOfferFromYaml = null;
			} finally {
				isProcessingYaml = false;
			}
		}
	});

	async function processYamlAndExtractCredentialOffer(yaml: string) {
		const res = await pb.send('api/credentials_issuers/get-credential-deeplink', {
			method: 'POST',
			body: {
				yaml
			}
		});
		const responseSchema = z.object({
			credentialOffer: z.string()
		});
		return responseSchema.parse(res);
	}
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
				<div class="flex h-60 w-60 items-center justify-center rounded-md border">
					<div class="text-muted-foreground text-sm">
						Processing YAML configuration...
					</div>
				</div>
			{:else}
				<!-- Show warning message if YAML processing failed -->
				{#if yamlProcessingError}
					<div
						class="mb-3 flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800"
					>
						<TriangleAlert class="h-4 w-4 shrink-0" />
						<div>
							<div class="font-medium">Dynamic generation failed</div>
						</div>
					</div>
				{/if}

				{#if !yamlProcessingError}
					<!-- Always use the same display format regardless of whether it's from YAML or default -->
					<QrCode src={qrLink} cellSize={10} class={['w-60 rounded-md']} />
					<div class="w-60 break-all pt-4 text-xs">
						<a href={qrLink} target="_self">{qrLink}</a>
					</div>
				{/if}
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

<EditSheet>
	{#snippet children({ closeSheet })}
		<T tag="h2" class="mb-4">{m.Edit()} {credential.name}</T>
		<EditCredentialForm
			{credential}
			{credentialIssuer}
			onSuccess={() => {
				closeSheet();
			}}
		/>
	{/snippet}
</EditSheet>
