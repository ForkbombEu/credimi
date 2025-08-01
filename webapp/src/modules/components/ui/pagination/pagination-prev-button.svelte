<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	// @ts-nocheck

	import { Pagination as PaginationPrimitive } from 'bits-ui';
	import ChevronLeft from 'lucide-svelte/icons/chevron-left';
	import { buttonVariants } from '@/components/ui/button/index.js';
	import { cn } from '@/components/ui/utils.js';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: PaginationPrimitive.PrevButtonProps = $props();
</script>

{#snippet Fallback()}
	<ChevronLeft class="size-4" />
	<span>Previous</span>
{/snippet}

<PaginationPrimitive.PrevButton
	bind:ref
	class={cn(
		buttonVariants({
			variant: 'ghost',
			class: 'gap-1 pl-2.5'
		}),
		className
	)}
	children={children || Fallback}
	{...restProps}
/>
