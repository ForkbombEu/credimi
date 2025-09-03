<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	// @ts-nocheck

	import { Pagination as PaginationPrimitive } from 'bits-ui';

	import { type Props, buttonVariants } from '@/components/ui/button/index.js';
	import { cn } from '@/components/ui/utils.js';

	let {
		ref = $bindable(null),
		class: className,
		size = 'icon',
		isActive = false,
		page,
		children,
		...restProps
	}: PaginationPrimitive.PageProps &
		Props & {
			isActive: boolean;
		} = $props();
</script>

{#snippet Fallback()}
	{page.value}
{/snippet}

<PaginationPrimitive.Page
	bind:ref
	{page}
	class={cn(
		buttonVariants({
			variant: isActive ? 'outline' : 'ghost',
			size
		}),
		className
	)}
	children={children || Fallback}
	{...restProps}
/>
