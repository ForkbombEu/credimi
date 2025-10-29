<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { pb } from '@/pocketbase/index.js';
	import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';
	import { Collections } from '@/pocketbase/types/index.generated.js';

	export async function getCredentialsDetails(itemId: string, fetchFn = fetch) {
		const credential = await new PocketbaseQueryAgent(
			{
				collection: 'credentials',
				expand: ['credential_issuer']
			},
			{ fetch: fetchFn }
		).getOne(itemId);

		const credentialIssuerMarketplaceEntry = await pb
			.collection('marketplace_items')
			.getFirstListItem(
				`id = '${credential.credential_issuer}' && type = '${Collections.CredentialIssuers}'`,
				{ fetch: fetchFn }
			);

		const credentialIssuer = credential.expand?.credential_issuer;
		if (!credentialIssuer) throw new Error('Credential issuer not found');

		return pageDetails('credentials', {
			credential,
			credentialIssuer,
			credentialIssuerMarketplaceEntry
		});
	}
</script>

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import type { CredentialConfiguration } from '$lib/types/openid.js';

	import { createIntentUrl } from '$lib/credentials/index.js';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import { MarketplaceItemCard } from '$lib/marketplace';
	import { generateDeeplinkFromYaml } from '$lib/utils';
	import { String } from 'effect';
	import { onMount } from 'svelte';

	import QrStateful from '@/qr/qr-stateful.svelte';

	import CodeSection from './_utils/code-section.svelte';
	import DescriptionSection from './_utils/description-section.svelte';
	import LayoutWithToc from './_utils/layout-with-toc.svelte';
	import PageSection from './_utils/page-section.svelte';
	import { sections as sec } from './_utils/sections';
	import { pageDetails } from './_utils/types';

	//

	type Props = Awaited<ReturnType<typeof getCredentialsDetails>>;
	let { credential, credentialIssuer, credentialIssuerMarketplaceEntry }: Props = $props();

	//

	let isProcessingYaml = $state(false);
	let yamlProcessingError = $state(false);
	let qrLink = $state<string>('');

	const credentialConfiguration = $derived(
		credential.json as CredentialConfiguration | undefined
	);

	onMount(async () => {
		if (credential.yaml && String.isNonEmpty(credential.yaml)) {
			isProcessingYaml = true;
			yamlProcessingError = false;
			try {
				const result = await generateDeeplinkFromYaml(credential.yaml);
				if (result.deeplink) {
					qrLink = result.deeplink;
				}
			} catch (error) {
				console.error('Failed to process YAML for credential offer:', error);
				yamlProcessingError = true;
				qrLink = createIntentUrl(credential, credentialIssuer.url);
			} finally {
				isProcessingYaml = false;
			}
		} else if (String.isNonEmpty(credential.deeplink)) {
			qrLink = credential.deeplink;
		} else {
			qrLink = createIntentUrl(credential, credentialIssuer.url);
		}
	});

	// Emptiness checks

	const signingAlgorithms = $derived(
		credentialConfiguration?.credential_signing_alg_values_supported?.join(', ')
	);

	const cryptographicBindingMethods = $derived(
		credentialConfiguration?.cryptographic_binding_methods_supported?.join(', ')
	);

	const isCredentialPropertiesEmpty = $derived(
		String.isEmpty(credential.format) &&
			String.isEmpty(credential.locale) &&
			!signingAlgorithms &&
			!cryptographicBindingMethods
	);

	//

	const sections: IndexItem[] = $derived.by(() => {
		const sections = [
			sec.credential_properties,
			sec.qr_code,
			sec.description,
			sec.credential_subjects,
			sec.compatible_issuer
		];
		if (credential.yaml) {
			sections.splice(4, 0, sec.workflow_yaml);
		}
		return sections;
	});
</script>

<LayoutWithToc {sections}>
	<div class="flex flex-col items-start gap-6 md:flex-row">
		<PageSection
			indexItem={sec.credential_properties}
			class="grow"
			empty={isCredentialPropertiesEmpty}
		>
			<div class="flex gap-6">
				<InfoBox label="Format" value={credential.format} />
				<InfoBox label="Locale" value={credential.locale} />
			</div>

			<div class="flex gap-6">
				<InfoBox label="Signing algorithms supported" value={signingAlgorithms} />
				<InfoBox
					label="Cryptographic binding methods supported"
					value={cryptographicBindingMethods}
				/>
			</div>
		</PageSection>

		<PageSection indexItem={sec.qr_code} class="flex flex-col items-stretch !space-y-0">
			<QrStateful
				src={qrLink}
				isLoading={isProcessingYaml}
				error={yamlProcessingError ? 'Dynamic generation failed' : undefined}
				loadingText="Processing YAML configuration..."
				placeholder="No credential offer available"
			/>

			{#if qrLink && !isProcessingYaml && !yamlProcessingError}
				<div class="w-60 break-all pt-4 text-xs">
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a href={qrLink} target="_blank">{qrLink}</a>
				</div>
			{/if}
		</PageSection>
	</div>

	<DescriptionSection description={credential.description} />

	<CodeSection
		indexItem={sec.credential_subjects}
		code={credentialConfiguration ? JSON.stringify(credentialConfiguration, null, 2) : null}
		language="json"
	/>

	{#if credential.yaml}
		<CodeSection indexItem={sec.workflow_yaml} code={credential.yaml} language="yaml" />
	{/if}

	<PageSection indexItem={sec.compatible_issuer}>
		<MarketplaceItemCard item={credentialIssuerMarketplaceEntry} />
	</PageSection>
</LayoutWithToc>

<!-- 
<EditSheet>
	{#snippet children({ closeSheet })}
		<T tag="h2" class="mb-4">{m.Edit()} {credential.display_name}</T>
		<EditCredentialForm
			{credential}
			{credentialIssuer}
			onSuccess={() => {
				closeSheet();
			}}
		/>
	{/snippet}
</EditSheet>
-->
