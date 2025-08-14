<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { Building2, Layers } from 'lucide-svelte';
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import { String } from 'effect';
	import { MarketplaceItemCard } from '../../../_utils/index.js';
	import MarketplacePageLayout from '$lib/layout/marketplace-page-layout.svelte';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';

	//

	let { data } = $props();
	const { credentialIssuer, credentialsMarketplaceItems } = $derived(data);

	//

	const sections = {
		general_info: {
			icon: Building2,
			anchor: 'general_info',
			label: m.General_info()
		},
		// svelte-ignore state_referenced_locally
		...(credentialIssuer?.description && {
			description: {
				icon: Layers,
				anchor: 'description',
				label: m.Description()
			}
		}),
		credentials: {
			icon: Layers,
			anchor: 'credentials',
			label: 'Supported credentials'
		}
	} satisfies Record<string, IndexItem>;
</script>

<MarketplacePageLayout tableOfContents={sections}>
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

	{#if credentialIssuer.description && sections.description}
		<div class="space-y-6">
			<PageHeader title={sections.description.label} id={sections.description.anchor} />

			<div class="prose">
				<RenderMd content={credentialIssuer.description} />
			</div>
		</div>
	{/if}

	<div class="space-y-6">
		<PageHeader title={sections.credentials.label} id={sections.credentials.anchor} />

		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			{#each credentialsMarketplaceItems as credential}
				<MarketplaceItemCard item={credential} />
			{:else}
				<div class="p-4 border border-black/20 rounded-md">
					<T class="text-center text-black/30">No credentials found</T>
				</div>
			{/each}
		</div>
	</div>
</MarketplacePageLayout>
