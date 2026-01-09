<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';

	import BackButton from '$lib/layout/back-button.svelte';
	import InfoBox from '$lib/layout/infoBox.svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageHeaderIndexed from '$lib/layout/pageHeaderIndexed.svelte';
	import PageIndex from '$lib/layout/pageIndex.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import { Building2, Layers } from 'lucide-svelte';

	import type { MarketplaceItemsResponse, OrganizationsResponse } from '@/pocketbase/types';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase/index.js';

	import { MarketplaceItemCard } from '../marketplace';

	//

	type Props = {
		organization: OrganizationsResponse;
		marketplaceItems: MarketplaceItemsResponse[];
		isPreview?: boolean;
	};

	let { organization, marketplaceItems, isPreview = false }: Props = $props();

	//

	const sections = {
		general_info: {
			icon: Building2,
			anchor: 'general_info',
			label: m.General_info()
		},
		services_and_products: {
			icon: Layers,
			anchor: 'services-and-products',
			label: m.Services_and_products()
		}
	} satisfies Record<string, IndexItem>;

	const organizationLogoUrl = $derived(pb.files.getURL(organization, organization.logo));
</script>

<PageTop contentClass="!space-y-4">
	{#if !isPreview}
		<BackButton href="/organizations">Back to organizations</BackButton>
	{/if}
	<div class="flex items-center gap-6">
		<Avatar
			src={organizationLogoUrl}
			class="size-24 rounded-sm border text-2xl"
			fallback={organization.name}
		/>

		<div class="space-y-3">
			<div class="space-y-1">
				<T class="text-sm">{m.Organization_name()}</T>
				<T tag="h1">{organization.name}</T>
			</div>
		</div>
	</div>
</PageTop>

<PageContent class="bg-secondary grow" contentClass="flex flex-col md:flex-row gap-16 items-start">
	<div class="sticky top-5 shrink-0">
		<PageIndex sections={Object.values(sections)} />
	</div>

	<div class="grow space-y-12">
		<div class="space-y-6">
			<PageHeaderIndexed indexItem={sections.general_info} />
			<div class="flex gap-6">
				<InfoBox label="Legal entity">{organization.legal_entity}</InfoBox>
				<InfoBox label="Country">{organization.country}</InfoBox>
			</div>
			<InfoBox label={m.Description()}>{organization.description}</InfoBox>

			<div class="flex gap-6">
				<InfoBox label="Website">
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a href={organization.external_website_url} target="_blank">
						{organization.external_website_url}
					</a>
				</InfoBox>
				<InfoBox label="Contact email">
					<a href={`mailto:${organization.contact_email}`} target="_blank">
						{organization.contact_email}
					</a>
				</InfoBox>
			</div>
		</div>

		<div>
			<PageHeaderIndexed indexItem={sections.services_and_products} />

			<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
				{#each marketplaceItems as item (item.id)}
					<MarketplaceItemCard {item} />
				{/each}
			</div>
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
