<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		tag?: 'h1' | 'h2' | 'h3' | 'h4' | 'p' | 'small';
		margin?: boolean;
		class?: string | undefined;
		children?: Snippet;
	}

	let { tag = 'p', margin = false, class: className = undefined, children }: Props = $props();
</script>

{#if tag == 'h1'}
	<h1 class="scroll-m-20 text-4xl font-extrabold tracking-tight lg:text-5xl {className}">
		{@render children?.()}
	</h1>
{:else if tag == 'h2'}
	<h2
		class="scroll-m-20 pb-2 text-3xl font-semibold tracking-tight transition-colors first:mt-0 {className}"
	>
		{@render children?.()}
	</h2>
{:else if tag == 'h3'}
	<h3 class="scroll-m-20 text-2xl font-semibold tracking-tight {className}">
		{@render children?.()}
	</h3>
{:else if tag == 'h4'}
	<h4 class="scroll-m-20 text-xl font-semibold tracking-tight {className}">
		{@render children?.()}
	</h4>
{:else if tag == 'p'}
	<p class="block {margin ? 'leading-7 [&:not(:first-child)]:mt-6' : ''} {className}">
		{@render children?.()}
	</p>
{:else if tag == 'small'}
	<small class="text-sm font-medium leading-none {className}">{@render children?.()}</small>
{:else if tag == 'huge'}
	<h1 class="scroll-m-20 text-5xl font-extrabold tracking-tight lg:text-6xl {className}">
		{@render children?.()}
	</h1>
{/if}
