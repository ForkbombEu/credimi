<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageIndex from '$lib/layout/pageIndex.svelte';
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import type { Snippet } from 'svelte';

	type Props = {
		tableOfContents?: Record<string, IndexItem>;
		children: Snippet;
	};

	let { tableOfContents, children }: Props = $props();
</script>

{#if tableOfContents}
	<PageIndex sections={Object.values(tableOfContents)} class="top-5 md:sticky" />
{/if}

<div class="grow space-y-16">
	{@render children()}
</div>
