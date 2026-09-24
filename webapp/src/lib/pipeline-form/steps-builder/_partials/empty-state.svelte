<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		text?: string;
		children?: Snippet;
		padded?: boolean;
	};

	let { text, children, padded = true }: Props = $props();
</script>

<div class={padded ? 'p-4' : undefined}>
	<div
		class="flex items-center justify-center gap-2 rounded-md bg-slate-100 p-4 text-sm text-muted-foreground"
	>
		{#if children}
			{@render children()}
		{:else if text}
			<T>{text}</T>
		{/if}
	</div>
</div>
