<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import OrganizationLayout from '@/organizations/components/organizationLayout.svelte';

	const { data } = $props();
	const { organization } = $derived(data);
</script>

<OrganizationLayout org={organization}>
	<pre>{JSON.stringify(organization, null, 2)}</pre>
</OrganizationLayout>
