<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CredentialIssuerSection from './_credentials/credential-issuer-section.svelte';
	import WalletsSection from './_wallets/wallets-section.svelte';
	import VerifiersSection from './_verifiers/verifiers-section.svelte';

	//

	let { data } = $props();
	const organizationId = $derived(data.organization?.id);
</script>

<div class="space-y-12">
	<div class="space-y-4">
		<CredentialIssuerSection {organizationId} />
	</div>

	<div class="space-y-4">
		<WalletsSection {organizationId} />
	</div>

	<div class="space-y-4">
		<VerifiersSection {organizationId} />
	</div>
</div>
