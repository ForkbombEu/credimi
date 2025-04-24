<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import PageIndex from '$lib/layout/pageIndex.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { Building2, Layers } from 'lucide-svelte';
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import { String } from 'effect';
	import CredentialCard from '$lib/layout/credentialCard.svelte';
	import BackButton from '$lib/layout/back-button.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import type { PocketbaseQueryResponse } from '@/pocketbase/query';

	//

	type Props = {
		credentialIssuer: PocketbaseQueryResponse<
			'credential_issuers',
			['credentials_via_credential_issuer']
		>;
	};

	let { credentialIssuer }: Props = $props();

	//

	const sections = {
		general_info: {
			icon: Building2,
			anchor: 'general_info',
			label: m.General_info()
		},
		credentials: {
			icon: Layers,
			anchor: 'credentials',
			label: 'Supported credentials'
		}
		// test_results: {
		// 	icon: FolderCheck,
		// 	anchor: 'test_results',
		// 	label: m.Test_results()
		// },
		// compatible_verifiers: {
		// 	icon: ScanEye,
		// 	anchor: 'compatible_verifiers',
		// 	label: m.Compatible_verifiers()
		// }
	} satisfies Record<string, IndexItem>;

	// //

	// let isClaimFormOpen = $state(false);

	const title = $derived(
		String.isNonEmpty(credentialIssuer.name) ? credentialIssuer.name : credentialIssuer.url
	);
</script>

<PageTop contentClass="!space-y-4">
	<BackButton href="/services">Back to services</BackButton>
	<div class="flex items-center gap-6">
		{#if credentialIssuer.logo_url}
			<Avatar src={credentialIssuer.logo_url} class="size-32 rounded-sm" hideIfLoadingError />
		{/if}

		<div class="space-y-3">
			<div class="space-y-1">
				<T class="text-sm">{m.Service_name()}</T>
				<T tag="h1">{title}</T>
			</div>
		</div>
	</div>
</PageTop>

<PageContent class="bg-secondary grow" contentClass="flex gap-12 items-start">
	<PageIndex sections={Object.values(sections)} class="sticky top-5" />

	<div class="grow space-y-16">
		<div class="space-y-6">
			<PageHeader title={sections.general_info.label} id={sections.general_info.anchor} />

			<InfoBox label="URL">
				<a href={credentialIssuer.url} class="hover:underline" target="_blank">
					{credentialIssuer.url}
				</a>
			</InfoBox>

			{#if String.isNonEmpty(credentialIssuer.repo_url)}
				<InfoBox label="Repository">
					<a href={credentialIssuer.repo_url} class="hover:underline" target="_blank">
						{credentialIssuer.repo_url}
					</a>
				</InfoBox>
			{/if}

			{#if String.isNonEmpty(credentialIssuer.homepage_url)}
				<InfoBox label="Homepage">
					<a href={credentialIssuer.homepage_url} class="hover:underline" target="_blank">
						{credentialIssuer.homepage_url}
					</a>
				</InfoBox>
			{/if}
		</div>

		<div class="space-y-6">
			<PageHeader title={sections.credentials.label} id={sections.credentials.anchor} />

			<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
				{#each credentialIssuer.expand?.credentials_via_credential_issuer ?? [] as credential}
					<CredentialCard {credential} />
				{:else}
					<div class="p-4 border border-black/20 rounded-md">
						<T class="text-center text-black/30">No credentials found</T>
					</div>
				{/each}
			</div>
		</div>
	</div>
</PageContent>
