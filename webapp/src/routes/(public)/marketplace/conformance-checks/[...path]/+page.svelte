<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CheckPage from './_partials/check-page.svelte';
	import CollectionPage from './_partials/collection-page.svelte';

	//

	let { data } = $props();
	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	let { conformanceChecks, organization, ...rest } = $derived(data);
</script>

{#if rest.type === 'collection-page'}
	<CollectionPage {...rest} />
{:else if rest.type === 'file-page'}
	<CheckPage {...rest} />
{/if}
