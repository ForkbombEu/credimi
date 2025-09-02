<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WithElementRef } from 'bits-ui';
	import type { HTMLAttributes } from 'svelte/elements';

	import { cn } from '@/components/ui/utils.js';

	let {
		ref = $bindable(null),
		class: className,
		level = 5,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & {
		level?: 1 | 2 | 3 | 4 | 5 | 6;
	} = $props();
</script>

<div
	role="heading"
	aria-level={level}
	bind:this={ref}
	class={cn('mb-1 font-medium leading-none tracking-tight', className)}
	{...restProps}
>
	{@render children?.()}
</div>
