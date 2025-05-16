<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import OrganizationPage from '$lib/pages/organization-page.svelte';

	//

	let { data } = $props();
	const { organization, marketplaceItems } = $derived(data);
</script>

<OrganizationPage {organization} {marketplaceItems} />
