<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { pb } from '@/pocketbase/index.js';
	import { Collections } from '@/pocketbase/types/index.generated.js';
	import { partitionPromises } from '@/utils/promise';

	export async function getCredentialsDetails(itemId: string, fetchFn = fetch) {
		const credential = await pb.collection('credentials').getOne(itemId, { fetch: fetchFn });

		// Handle potentially missing credential issuer
		const [credentialIssuerMarketplaceEntries] = await partitionPromises([
			pb
				.collection('marketplace_items')
				.getFirstListItem(
					`id = '${credential.credential_issuer}' && type = '${Collections.CredentialIssuers}'`,
					{ fetch: fetchFn }
				)
		]);
		const credentialIssuerMarketplaceEntry = credentialIssuerMarketplaceEntries[0] ?? null;

		return pageDetails('credentials', {
			credential,
			credentialIssuerMarketplaceEntry
		});
	}
</script>

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import type { CredentialConfiguration } from '$lib/types/openid.js';

	import NotFoundCard from '$lib/components/not-found-card.svelte';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import { MarketplaceItemCard } from '$lib/marketplace';
	import { String } from 'effect';

	import CodeSection from './_utils/code-section.svelte';
	import DescriptionSection from './_utils/description-section.svelte';
	import LayoutWithToc from './_utils/layout-with-toc.svelte';
	import PageSection from './_utils/page-section.svelte';
	import { sections as sec } from './_utils/sections';
	import { pageDetails } from './_utils/types';
	import QrSection from './qr-section.svelte';

	//

	type Props = Awaited<ReturnType<typeof getCredentialsDetails>>;
	let { credential, credentialIssuerMarketplaceEntry }: Props = $props();

	//

	const credentialConfiguration = $derived(
		credential.json as CredentialConfiguration | undefined
	);

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

		<QrSection yaml={credential.yaml} deeplink={credential.deeplink} />
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
		{#if credentialIssuerMarketplaceEntry}
			<MarketplaceItemCard item={credentialIssuerMarketplaceEntry} />
		{:else}
			<NotFoundCard />
		{/if}
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
