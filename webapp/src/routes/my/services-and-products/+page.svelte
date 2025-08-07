<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CredentialIssuerSection from './_credentials/credential-issuer-section.svelte';
	import WalletsSection from './_wallets/wallets-section.svelte';
	import VerifiersSection from './_verifiers/verifiers-section.svelte';
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import PageIndex from '$lib/layout/pageIndex.svelte';
	import { m } from '@/i18n';
	import SmoothPageScroll from '$lib/layout/smooth-page-scroll.svelte';

	//

	let { data } = $props();
	const organizationId = $derived(data.organization?.id ?? '');

	//

	const tableOfContents = [
		{
			label: m.Credential_issuers(),
			anchor: 'credential-issuers'
		},
		{
			label: m.Wallets(),
			anchor: 'wallets'
		},
		{
			label: m.Verifiers(),
			anchor: 'verifiers'
		}
	] satisfies Array<IndexItem>;
</script>

<SmoothPageScroll />

<div class="flex flex-col items-start gap-14 md:flex-row md:gap-10">
	<div class="relative top-5 self-stretch md:sticky md:self-start">
		<PageIndex title={m.Sections()} sections={tableOfContents} class="top-5 md:sticky" />
	</div>
	<div class="grow space-y-12 self-stretch md:self-start">
		<div class="space-y-4">
			<CredentialIssuerSection {organizationId} id={tableOfContents[0].anchor} />
		</div>

		<div class="space-y-4">
			<WalletsSection
				{organizationId}
				workflows={data.workflows}
				id={tableOfContents[1].anchor}
			/>
		</div>

		<div class="space-y-4">
			<VerifiersSection {organizationId} id={tableOfContents[2].anchor} />
		</div>
	</div>
</div>
