<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';

	import PageIndex from '$lib/layout/pageIndex.svelte';
	import SmoothPageScroll from '$lib/layout/smooth-page-scroll.svelte';

	import { m } from '@/i18n';

	import CredentialIssuerSection from './_credentials/credential-issuer-section.svelte';
	import VerifiersSection from './_verifiers/verifiers-section.svelte';
	import WalletsSection from './_wallets/wallets-section.svelte';

	//

	let { data } = $props();
	const organizationId = $derived(data.organization?.id ?? '');
	const organization = $derived(data.organization);

	//

	const tableOfContents = {
		credential_issuers: {
			label: m.Credential_issuers(),
			anchor: 'credential-issuers'
		},
		wallets: {
			label: m.Wallets(),
			anchor: 'wallets'
		},
		verifiers: {
			label: m.Verifiers(),
			anchor: 'verifiers'
		}
	} satisfies Record<string, IndexItem>;
</script>

<SmoothPageScroll />

<div class="flex flex-col items-start gap-14 md:flex-row md:gap-10">
	<div class="relative top-5 self-stretch md:sticky md:self-start">
		<PageIndex
			title={m.Sections()}
			sections={Object.values(tableOfContents)}
			class="top-5 md:sticky"
		/>
	</div>
	<div class="grow space-y-12 self-stretch md:self-start">
		<div class="space-y-4">
			<CredentialIssuerSection
				{organizationId}
				{organization}
				id={tableOfContents.credential_issuers.anchor}
			/>
		</div>

		<div class="space-y-4">
			<WalletsSection {organizationId} {organization} id={tableOfContents.wallets.anchor} />
		</div>

		<div class="space-y-4">
			<VerifiersSection
				{organizationId}
				{organization}
				id={tableOfContents.verifiers.anchor}
			/>
		</div>
	</div>
</div>
