<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { PencilIcon } from '@lucide/svelte';
	import { userOrganization } from '$lib/app-state';
	import { getHubItemData } from '$lib/hub';
	import { hubItemToSectionHref } from '$lib/hub/utils';
	import { getPath } from '$lib/utils/index.js';

	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import CredentialIssuerPage from './_partials/credential-issuer-page.svelte';
	import CredentialPage from './_partials/credential-page.svelte';
	import HubPageTop from './_partials/hub-page-top.svelte';
	import PipelinePage from './_partials/pipeline-page.svelte';
	import UseCaseVerificationPage from './_partials/use-case-verification-page.svelte';
	import VerifierPage from './_partials/verifier-page.svelte';
	import WalletPage from './_partials/wallet-page.svelte';

	//

	let { data } = $props();
	const { hubItem, pageDetails } = $derived(data);

	const { logo, display } = $derived(getHubItemData(hubItem));

	const isCurrentUserOwner = $derived(
		userOrganization.current?.id === hubItem.organization_id
	);
</script>

<!-- Owner edit topbar -->

{#if isCurrentUserOwner}
	<div class="border-t-2 border-t-primary bg-secondary py-2">
		<div
			class="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-3 px-4 text-sm md:px-8"
		>
			<T>{m.This_item_is_yours({ item: display.labels.singular })}</T>
			<div class="flex items-center gap-3">
				<T>{m.Last_edited()}: {new Date(hubItem.updated).toLocaleDateString()}</T>
				<Button
					size="sm"
					class="h-8! text-xs"
					href={hubItemToSectionHref(hubItem)}
				>
					<PencilIcon />
					{m.Make_changes()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- General page content -->

<HubPageTop
	title={hubItem.name}
	textToCopy={getPath(hubItem)}
	badge={display}
	hideTopBorder={isCurrentUserOwner}
	{logo}
	linkAboveTitle={{
		href: `/organizations/${hubItem.organization_canonified_name}`,
		title: hubItem.organization_name
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
