<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLTdAttributes } from 'svelte/elements';

	type Props = {
		/**
		 * Text cell: non-empty string is shown; otherwise the fallback.
		 * Prefer this over `when`/`children` for simple values.
		 */
		value?: string | null | undefined;
		/**
		 * Content cell: when true, render `children`; when false, render the fallback.
		 * Ignored when `value` is passed. If omitted and `children` exist, children render.
		 */
		when?: boolean;
		/** Shown when there is no content. Default dash; use `empty` for blank cells. */
		fallback?: 'dash' | 'empty';
		/** Leading expand column: shrink-wrap, no horizontal pad. */
		expand?: boolean;
		/** Status / child-name nest: uses `--nest-pl` for padding-left. */
		nest?: boolean;
		class?: string;
		colspan?: number;
		children?: Snippet;
	} & Omit<HTMLTdAttributes, 'children' | 'colspan' | 'class'>;

	let {
		value,
		when,
		fallback = 'dash',
		expand = false,
		nest = false,
		class: className,
		colspan,
		children,
		...rest
	}: Props = $props();

	const showContent = $derived(
		value !== undefined ? Boolean(value) : when !== undefined ? when : Boolean(children)
	);
</script>

<td
	class={[
		expand
			? 'w-px px-0 py-(--td-py) whitespace-nowrap'
			: nest
				? 'py-(--td-py) pr-(--td-px) pl-(--nest-pl)'
				: 'px-(--td-px) py-(--td-py)',
		className
	]}
	{colspan}
	{...rest}
>
	{#if showContent}
		{#if value !== undefined}
			{value}
		{:else}
			{@render children?.()}
		{/if}
	{:else if fallback === 'dash'}
		<span class="text-muted-foreground opacity-50">—</span>
	{/if}
</td>
