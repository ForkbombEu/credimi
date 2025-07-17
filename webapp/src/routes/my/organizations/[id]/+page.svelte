<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import OrganizationLayout from '@/organizations/components/organizationLayout.svelte';
	import CopyableCodeBlock from '$lib/layout/copyableCodeBlock.svelte';

	const { data } = $props();
	const { organization } = $derived(data);
</script>

<OrganizationLayout org={organization}>
	<CopyableCodeBlock content={JSON.stringify(organization, null, 2)} language="json" />
</OrganizationLayout>
