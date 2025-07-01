<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		right?: Snippet;
		left?: Snippet;
	};

	let { right, left }: Props = $props();
</script>

<div class="flex flex-col items-stretch gap-4 md:flex-row">
	{#if left}
		<div class="shrink-0 grow basis-1">
			{@render left()}
		</div>
	{/if}
	{#if right}
		<div class="shrink-0 grow basis-1">
			{@render right()}
		</div>
	{/if}
</div>
