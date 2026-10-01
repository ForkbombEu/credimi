<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export type CellDensity = 'compact' | 'comfortable';

	/** Shared td padding for the pipeline executions table. */
	export function cellPadClass(
		density: CellDensity,
		opts: { child?: boolean; expand?: boolean } = {}
	): string {
		if (opts.expand) {
			// Shrink-wrap to the control; nest indent uses inline padding-left only.
			const y = density === 'compact' ? 'py-0.5' : opts.child ? 'py-1' : 'py-2';
			return `w-px whitespace-nowrap px-0 ${y}`;
		}
		if (density === 'compact') return 'px-2 py-0.5';
		if (opts.child) return 'px-3 py-1 text-xs';
		return 'px-3 py-2';
	}
</script>

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
		 * Ignored when `value` is passed.
		 */
		when?: boolean;
		/** Shown when there is no content. Default dash; use `empty` for blank cells. */
		fallback?: 'dash' | 'empty';
		density?: CellDensity;
		/** Comfortable child rows use tighter vertical padding. */
		child?: boolean;
		/** Leading expand column: narrower horizontal padding. */
		expand?: boolean;
		class?: string;
		colspan?: number;
		children?: Snippet;
	} & Omit<HTMLTdAttributes, 'children' | 'colspan' | 'class'>;

	let {
		value,
		when,
		fallback = 'dash',
		density = 'comfortable',
		child = false,
		expand = false,
		class: className,
		colspan,
		children,
		...rest
	}: Props = $props();

	const showContent = $derived(value !== undefined ? Boolean(value) : when === true);
	const padClass = $derived(cellPadClass(density, { child, expand }));
</script>

<td class={[padClass, className]} {colspan} {...rest}>
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
