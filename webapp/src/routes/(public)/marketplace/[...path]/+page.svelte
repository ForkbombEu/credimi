<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { userOrganization } from '$lib/app-state';
	import PageTop from '$lib/layout/pageTop.svelte';
	import { getMarketplaceItemData, MarketplaceItemTypeDisplay } from '$lib/marketplace';
	import { marketplaceItemToSectionHref } from '$lib/marketplace/sections';
	import { ArrowLeft, PencilIcon } from 'lucide-svelte';

	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import CredentialIssuerPage from './_partials/credential-issuer-page.svelte';
	import CredentialPage from './_partials/credential-page.svelte';
	import UseCaseVerificationPage from './_partials/use-case-verification-page.svelte';
	import VerifierPage from './_partials/verifier-page.svelte';
	import WalletPage from './_partials/wallet-page.svelte';

	//

	let { data } = $props();
	const { marketplaceItem, pageDetails } = $derived(data);

	const { logo, display } = $derived(getMarketplaceItemData(marketplaceItem));

	const isCurrentUserOwner = $derived(
		userOrganization.current?.id === marketplaceItem.organization_id
	);
</script>

<!-- Owner edit topbar -->

{#if isCurrentUserOwner}
	<div class="border-t-primary border-t-2 bg-[#E2DCF8] py-2">
		<div
			class="mx-auto flex max-w-screen-xl flex-wrap items-center justify-between gap-3 px-4 text-sm md:px-8"
		>
			<T>{m.This_item_is_yours({ item: display.label })}</T>
			<div class="flex items-center gap-3">
				<T>{m.Last_edited()}: {new Date(marketplaceItem.updated).toLocaleDateString()}</T>
				<Button
					size="sm"
					class="!h-8 text-xs"
					href={marketplaceItemToSectionHref(marketplaceItem)}
				>
					<PencilIcon />
					{m.Make_changes()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- General page content -->

<PageTop hideTopBorder={isCurrentUserOwner} contentClass="!space-y-4">
	<Button variant="link" class="gap-1 p-0" href="/marketplace">
		<ArrowLeft />
		{m.Back()}
	</Button>

	<div class="flex items-center gap-6">
		{#if logo}
			<Avatar src={logo} class="size-32 rounded-md border" hideIfLoadingError />
		{/if}

		<div class="space-y-3">
			<div>
				<div class="space-y-1">
					<A
						class="block"
						href="/organizations/{marketplaceItem.organization_canonified_name}"
					>
						{marketplaceItem.organization_name}
					</A>
					<div class="flex items-center gap-2">
						<T tag="h1">{marketplaceItem.name}</T>
						<CopyButtonSmall
							textToCopy={marketplaceItem.path}
							square
							variant="ghost"
							size="xs"
						/>
					</div>
				</div>

				{#if display}
					<div class="pt-4">
						<MarketplaceItemTypeDisplay data={display} />
					</div>
				{/if}
			</div>
		</div>
	</div>
</PageTop>

<!-- Type-specific page -->

{#if pageDetails.type == 'credential_issuers'}
	<CredentialIssuerPage {...pageDetails} />
{:else if pageDetails.type == 'credentials'}
	<CredentialPage {...pageDetails} />
{:else if pageDetails.type == 'wallets'}
	<WalletPage {...pageDetails} />
{:else if pageDetails.type == 'verifiers'}
	<VerifierPage {...pageDetails} />
{:else if pageDetails.type == 'use_cases_verifications'}
	<UseCaseVerificationPage {...pageDetails} />
{/if}
