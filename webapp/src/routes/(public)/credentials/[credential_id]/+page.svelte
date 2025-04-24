<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BackButton from '$lib/layout/back-button.svelte';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import PageIndex from '$lib/layout/pageIndex.svelte';
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import ServiceCard from '$lib/layout/serviceCard.svelte';
	import type { CredentialConfiguration } from '$lib/types/openid.js';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n/index.js';
	import { QrCode } from '@/qr/index.js';
	import { Building2, FolderCheck, Layers3 } from 'lucide-svelte';
	import { String } from 'effect';

	let { data } = $props();
	const { credential } = $derived(data);

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
		// compatible_apps: {
		// 	icon: Building2,
		// 	anchor: 'compatible_apps',
		// 	label: 'Compatible apps'
		// },
		compatible_issuer: {
			icon: FolderCheck,
			anchor: 'compatible_issuer',
			label: 'Compatible issuer'
		}
		// test_results: {
		// 	icon: ScanEye,
		// 	anchor: 'test_results',
		// 	label: 'Test results'
		// }
	} satisfies Record<string, IndexItem>;

	const credentialConfiguration = $derived(
		credential.json as CredentialConfiguration | undefined
	);

	const credentialIssuer = $derived(credential.expand?.credential_issuer);

	function createIntentUrl(issuer: string | undefined, type: string): string {
		const data = {
			credential_configuration_ids: [type],
			credential_issuer: issuer
		};
		const credentialOffer = encodeURIComponent(JSON.stringify(data));
		return `openid-credential-offer://?credential_offer=${credentialOffer}`;
	}
</script>

<PageTop contentClass="!space-y-4">
	<BackButton href="/credentials">Back to credentials</BackButton>
	<div class="flex items-center gap-6">
		{#if credential.logo}
			<Avatar src={credential.logo} class="size-32 rounded-sm" hideIfLoadingError />
		{/if}

		<div class="space-y-3">
			<div class="space-y-1">
				<T class="text-sm">{m.Credential_name()}</T>
				<T tag="h1">{credential.name}</T>
			</div>
		</div>
	</div>
</PageTop>

<PageContent class="grow bg-secondary" contentClass="flex gap-12 items-start">
	<PageIndex sections={Object.values(sections)} class="sticky top-5" />

	<div class="grow space-y-16">
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

			<div class="flex flex-col items-center">
				<PageHeader title="Credential offer" id="qr" />
				<QrCode
					src={String.isNonEmpty(credential.deeplink)
						? credential.deeplink
						: createIntentUrl(credentialIssuer?.url, credential.type)}
					cellSize={10}
					class={['w-60 rounded-md']}
				/>
				<div class="w-60 break-all pt-4 text-xs">
					<a
						href={String.isNonEmpty(credential.deeplink)
							? credential.deeplink
							: createIntentUrl(credentialIssuer?.url, credential.type)}
						target="_self"
					>
						{String.isNonEmpty(credential.deeplink)
							? credential.deeplink
							: createIntentUrl(credentialIssuer?.url, credential.type)}</a
					>
				</div>
			</div>
		</div>

		<div class="space-y-6">
			<PageHeader
				title={sections.credential_subjects.label}
				id={sections.credential_subjects.anchor}
			/>

			{#if credentialConfiguration}
				<pre
					class="w-fit max-w-screen-lg overflow-x-clip rounded-xl border border-primary bg-card p-6 text-xs text-card-foreground shadow-sm ring-primary transition-transform hover:-translate-y-2 hover:ring-2">{JSON.stringify(
						credentialConfiguration,
						null,
						2
					)}</pre>
			{/if}
		</div>

		<!-- <div>
			<PageHeader
				title={sections.compatible_apps.label}
				id={sections.compatible_apps.anchor}
			/>
		</div> -->

		<div>
			<PageHeader
				title={sections.compatible_issuer.label}
				id={sections.compatible_issuer.anchor}
			/>

			{#if credentialIssuer}
				<ServiceCard service={credentialIssuer} />
			{/if}
		</div>
	</div>
</PageContent>
