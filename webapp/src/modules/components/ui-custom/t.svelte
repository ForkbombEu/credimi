<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	interface Props {
		tag?: 'h1' | 'h2' | 'h3' | 'h4' | 'p' | 'small' | 'huge';
		margin?: boolean;
		class?: ClassValue;
		children?: Snippet;
	}

	let { tag = 'p', margin = false, class: classValue, children }: Props = $props();
</script>

{#if tag == 'h1'}
	<h1 class={['scroll-m-20 text-4xl font-extrabold tracking-tight lg:text-5xl', classValue]}>
		{@render children?.()}
	</h1>
{:else if tag == 'h2'}
	<h2
		class={[
			'scroll-m-20 pb-2 text-3xl font-semibold tracking-tight transition-colors first:mt-0',
			classValue
		]}
	>
		{@render children?.()}
	</h2>
{:else if tag == 'h3'}
	<h3 class={['scroll-m-20 text-2xl font-semibold tracking-tight', classValue]}>
		{@render children?.()}
	</h3>
{:else if tag == 'h4'}
	<h4 class={['scroll-m-20 text-xl font-semibold tracking-tight', classValue]}>
		{@render children?.()}
	</h4>
{:else if tag == 'p'}
	<p class={['block', margin ? 'leading-7 [&:not(:first-child)]:mt-6' : '', classValue]}>
		{@render children?.()}
	</p>
{:else if tag == 'small'}
	<small class={['text-sm font-medium leading-none', classValue]}>{@render children?.()}</small>
{:else if tag == 'huge'}
	<h1 class={['scroll-m-20 text-5xl font-extrabold tracking-tight lg:text-6xl', classValue]}>
		{@render children?.()}
	</h1>
{/if}
