<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		children?: Snippet;
	};

	const { children }: Props = $props();
</script>

{#if children}
	<div class="bg-secondary/40 rounded-md p-2">
		<p class="text-sm font-bold">{@render children()}</p>
	</div>
{/if}
