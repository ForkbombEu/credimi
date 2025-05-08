<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import PageIndex from '$lib/layout/pageIndex.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { Building2, Layers, ScanEye } from 'lucide-svelte';
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import { pb } from '@/pocketbase/index.js';
	import type { OrganizationInfoResponse } from '@/pocketbase/types';
	import BackButton from '$lib/layout/back-button.svelte';

	type Props = {
		organizationInfo: OrganizationInfoResponse;
	};

	let { organizationInfo }: Props = $props();

	//

	const sections: Record<string, IndexItem> = {
		general_info: {
			icon: Building2,
			anchor: 'general_info',
			label: m.General_info()
		},
		apps: {
			icon: Layers,
			anchor: 'apps',
			label: m.Apps()
		},
		issuers: {
			icon: ScanEye,
			anchor: 'issuers',
			label: m.Issuers()
		}
	};
</script>

<PageTop contentClass="!space-y-4">
	<BackButton href="/organizations">Back to organizations</BackButton>
	<div class="flex items-center gap-6">
		{#if organizationInfo.logo}
			{@const providerUrl = pb.files.getURL(organizationInfo, organizationInfo.logo)}
			<Avatar src={providerUrl} class="size-32 rounded-sm" hideIfLoadingError />
		{/if}

		<div class="space-y-3">
			<div class="space-y-1">
				<T class="text-sm">{m.Organization_name()}</T>
				<T tag="h1">{organizationInfo.name}</T>
			</div>
		</div>
	</div>
</PageTop>

<PageContent class="bg-secondary grow" contentClass="flex flex-col md:flex-row gap-16 items-start">
	<div class="sticky top-5 shrink-0">
		<PageIndex sections={Object.values(sections)} />
	</div>

	<div class="max-w-prose grow space-y-12">
		<div class="space-y-6">
			<PageHeader title={sections.general_info.label} id={sections.general_info.anchor} />
			<div class="flex gap-6">
				<InfoBox label="Legal entity">{organizationInfo.legal_entity}</InfoBox>
				<InfoBox label="Country">{organizationInfo.country}</InfoBox>
			</div>
			<InfoBox label={m.Description()}>{organizationInfo.description}</InfoBox>

			<div class="flex gap-6">
				<InfoBox label="Website">
					<a href={organizationInfo.external_website_url} target="_blank">
						{organizationInfo.external_website_url}
					</a>
				</InfoBox>
				<InfoBox label="Contact email">
					<a href={`mailto:${organizationInfo.contact_email}`} target="_blank">
						{organizationInfo.contact_email}
					</a>
				</InfoBox>
			</div>
		</div>

		<div>
			<PageHeader title={m.Apps()} id="apps"></PageHeader>
		</div>

		<!-- TODO - Replace with MarketplaceItemCards -->
		<!--<div>
			<PageHeader title={m.Issuers()} id="issuers" />

			{#await credentialIssuersPromise then credential_issuers}
				<div class="space-y-2">
					{#each credential_issuers as issuer, index (issuer.id)}
						< service={issuer} />
					{:else}
						<div class="p-4 border border-black/20 rounded-md">
							<T class="text-center text-black/30">No issuers found</T>
						</div>
					{/each}
				</div>
			{/await}
		</div> -->
	</div>
</PageContent>

{#snippet CircledNumber(index: number)}
	<div
		class="border-primary flex size-4 shrink-0 items-center justify-center rounded-full border text-sm text-slate-500"
	>
		<p>
			{index}
		</p>
	</div>
{/snippet}
