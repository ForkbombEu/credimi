<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import InfoBox from '$lib/layout/infoBox.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import type { CredentialConfiguration } from '$lib/types/openid.js';
	import { QrCode } from '@/qr/index.js';
	import { Building2, FolderCheck, Layers3 } from 'lucide-svelte';
	import { String } from 'effect';
	import { MarketplaceItemCard } from '../../../_utils/index.js';
	import MarketplacePageLayout from '$lib/layout/marketplace-page-layout.svelte';
	import { createIntentUrl } from '$lib/credentials/index.js';
	import CopyableCodeBlock from '$lib/layout/copyableCodeBlock.svelte';

	let { data } = $props();
	const { credential, credentialIssuer, credentialIssuerMarketplaceEntry } = $derived(data);

	const sections = {
		credential_properties: {
			icon: Building2,
			anchor: 'credential_properties',
			label: 'Credential properties'
		},
		credential_subjects: {
			icon: Layers3,
			anchor: 'credential_subject',
			label: 'Credential subject'
		},
		compatible_issuer: {
			icon: FolderCheck,
			anchor: 'compatible_issuer',
			label: 'Compatible issuer'
		}
	} satisfies Record<string, IndexItem>;

	const credentialConfiguration = $derived(
		credential.json as CredentialConfiguration | undefined
	);

	const qrLink = $derived(
		String.isNonEmpty(credential.deeplink)
			? credential.deeplink
			: createIntentUrl(credential, credentialIssuer.url)
	);
</script>

<MarketplacePageLayout tableOfContents={sections}>
	<div class="flex items-start gap-6">
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

			<InfoBox label="Description" value={credential.description} />
			<InfoBox label="Type" value={credential.type} />
		</div>

		<div class="flex flex-col items-stretch">
			<PageHeader title="Credential offer" id="qr" />
			<QrCode src={qrLink} cellSize={10} class={['w-60 rounded-md']} />
			<div class="w-60 break-all pt-4 text-xs">
				<a href={qrLink} target="_self">{qrLink}</a>
			</div>
		</div>
	</div>

	<div class="space-y-6">
		<PageHeader
			title={sections.credential_subjects.label}
			id={sections.credential_subjects.anchor}
		/>

		{#if credentialConfiguration}
			<CopyableCodeBlock
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
