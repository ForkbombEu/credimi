<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		text?: string;
		children?: Snippet;
		containerClass?: ClassValue;
	};

	let { text, children, containerClass }: Props = $props();
</script>

<div class={['p-4', containerClass]}>
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
