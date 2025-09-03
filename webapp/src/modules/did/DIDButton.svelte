<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Button from '@/components/ui-custom/button.svelte';
	import { getUserDidUrl } from '@/did';
	import { featureFlags } from '@/features';
	import { m } from '@/i18n';

	let url: string | undefined = $state(undefined);

	getUserDidUrl().then((res) => {
		url = res;
	});
</script>

{#if $featureFlags.DID && url}
	<Button href={url} target="_blank" size="sm" variant="outline">
		{m.my_DID()}
	</Button>
{/if}
