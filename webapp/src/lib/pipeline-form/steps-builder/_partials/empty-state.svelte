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
	};

	let { text, children }: Props = $props();
</script>

<div class="p-4">
	<div
		class="text-muted-foreground flex items-center justify-center gap-2 rounded-md bg-slate-100 p-4 text-sm"
	>
		{#if children}
			{@render children()}
		{:else if text}
			<T>{text}</T>
		{/if}
	</div>
</div>
