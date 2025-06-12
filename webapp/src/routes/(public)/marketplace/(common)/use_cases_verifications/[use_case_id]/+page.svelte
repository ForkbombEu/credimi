<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import MarketplacePageLayout from '$lib/layout/marketplace-page-layout.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import { type IndexItem } from '$lib/layout/pageIndex.svelte';
	import { m } from '@/i18n/index.js';
	import { Building2, Key, Layers3 } from 'lucide-svelte';
	import MarketplaceItemCard from '$marketplace/_utils/marketplace-item-card.svelte';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import { String } from 'effect';
	import T from '@/components/ui-custom/t.svelte';
	import { QrCode } from '@/qr';

	//

	let { data } = $props();

	//

	const sections = {
		general_info: {
			icon: Building2,
			anchor: 'general_info',
			label: m.General_info()
		},
		related_verifier: {
			icon: Layers3,
			anchor: 'related_verifier',
			label: m.Related_verifier()
		},
		related_credentials: {
			icon: Key,
			anchor: 'related_credentials',
			label: m.Related_credentials()
		}
	} satisfies Record<string, IndexItem>;
</script>

<MarketplacePageLayout tableOfContents={sections}>
	<div class="flex flex-col gap-6 md:flex-row">
		<div class="grow space-y-6">
			<PageHeader title={sections.general_info.label} id={sections.general_info.anchor} />
			{#if String.isNonEmpty(data.useCaseVerification.description)}
				<RenderMd content={data.useCaseVerification.description} />
			{:else}
				<div class="rounded-md border border-black/20 p-4">
					<T class="text-center text-black/30">No description found</T>
				</div>
			{/if}
		</div>

		<div class="flex flex-col items-stretch">
			<PageHeader title={m.QR_code()} id="qr" />
			<QrCode src={data.useCaseVerification.deeplink} class="size-60 rounded-md" />
			<div class="break-all pt-2 font-mono text-xs">
				<a href={data.useCaseVerification.deeplink} class="hover:underline" target="_blank">
					{data.useCaseVerification.deeplink}
				</a>
			</div>
		</div>
	</div>

	<div class="flex w-full flex-col gap-6 sm:flex-row">
		<div class="shrink-0 grow basis-1">
			<PageHeader
				title={sections.related_verifier.label}
				id={sections.related_verifier.anchor}
			/>
			<MarketplaceItemCard item={data.verifierMarketplaceItem} />
		</div>

		<div class="shrink-0 grow basis-1">
			<PageHeader
				title={sections.related_credentials.label}
				id={sections.related_credentials.anchor}
			/>

			<div class="flex flex-col gap-2">
				{#each data.marketplaceCredentials as marketplaceCredential}
					<MarketplaceItemCard item={marketplaceCredential} />
				{/each}
			</div>
		</div>
	</div>
</MarketplacePageLayout>
