<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WithElementRef } from 'bits-ui';
	import type { Snippet } from 'svelte';
	import type { HTMLAnchorAttributes } from 'svelte/elements';

	import { cn } from '@/components/ui/utils.js';
	import { localizeHref } from '@/i18n';

	let {
		ref = $bindable(null),
		class: className,
		href = undefined,
		child,
		children,
		...restProps
	}: WithElementRef<HTMLAnchorAttributes> & {
		child?: Snippet<[{ props: HTMLAnchorAttributes }]>;
	} = $props();

	const attrs = $derived({
		class: cn('hover:text-foreground transition-colors', className),
		href: href ? localizeHref(href) : undefined,
		...restProps
	});
</script>

{#if child}
	{@render child({ props: attrs })}
{:else}
	<a bind:this={ref} {...attrs}>
		{@render children?.()}
	</a>
{/if}
