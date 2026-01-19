<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { userOrganization } from '$lib/app-state';
	import { getMarketplaceItemData } from '$lib/marketplace';
	import { marketplaceItemToSectionHref } from '$lib/marketplace/utils';
	import { PencilIcon } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import CredentialIssuerPage from './_partials/credential-issuer-page.svelte';
	import CredentialPage from './_partials/credential-page.svelte';
	import MarketplacePageTop from './_partials/marketplace-page-top.svelte';
	import PipelinePage from './_partials/pipeline-page.svelte';
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
			<T>{m.This_item_is_yours({ item: display.labels.singular })}</T>
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

<MarketplacePageTop
	title={marketplaceItem.name}
	textToCopy={marketplaceItem.path}
	badge={display}
	hideTopBorder={isCurrentUserOwner}
	{logo}
	linkAboveTitle={{
		href: `/organizations/${marketplaceItem.organization_canonified_name}`,
		title: marketplaceItem.organization_name
	}}
/>

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
{:else if pageDetails.type == 'pipelines'}
	<PipelinePage {...pageDetails} />
{/if}
