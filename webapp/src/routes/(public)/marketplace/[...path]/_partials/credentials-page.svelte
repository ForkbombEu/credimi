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
	import type { CredentialConfiguration } from '$lib/types/openid.js';

	import { createIntentUrl } from '$lib/credentials/index.js';
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import PageHeaderIndexed from '$lib/layout/pageHeaderIndexed.svelte';
	import { generateDeeplinkFromYaml } from '$lib/utils';
	import { MarketplaceItemCard } from '$marketplace/_utils';
	import EditCredentialForm from '$routes/my/services-and-products/_credentials/credential-form.svelte';
	import { String } from 'effect';
	import { onMount } from 'svelte';

	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import QrStateful from '@/qr/qr-stateful.svelte';

	import EditSheet from './edit-sheet.svelte';
	import LayoutWithToc from './layout-with-toc.svelte';
	import { sections as sec } from './sections';
	import { pageDetails } from './types';

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
</script>

<LayoutWithToc
	sections={[
		sec.credential_properties,
		sec.description,
		sec.credential_subjects,
		sec.compatible_issuer
	]}
>
	<div class="flex flex-col items-start gap-6 md:flex-row">
		<div class="grow space-y-6">
			<PageHeaderIndexed indexItem={sec.credential_properties} />

			<div class="flex gap-6">
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
		</div>

		<div class="flex flex-col items-stretch">
			<PageHeader title="Credential offer" id="qr" />

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
		</div>
	</div>

	<div class="space-y-6">
		<PageHeaderIndexed indexItem={sec.description} />
		<div class="prose">
			<RenderMd content={credential.description} />
		</div>
	</div>

	<div class="space-y-6">
		<PageHeaderIndexed indexItem={sec.credential_subjects} />

		{#if credentialConfiguration}
			<CodeDisplay
				content={JSON.stringify(credentialConfiguration, null, 2)}
				language="json"
				class="border-primary bg-card text-card-foreground ring-primary w-fit max-w-screen-lg overflow-x-clip rounded-xl border p-6 text-xs shadow-sm transition-transform hover:-translate-y-2 hover:ring-2"
			/>
		{/if}
	</div>

	<div>
		<PageHeaderIndexed indexItem={sec.compatible_issuer} />
		<MarketplaceItemCard item={credentialIssuerMarketplaceEntry} />
	</div>
</LayoutWithToc>

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
