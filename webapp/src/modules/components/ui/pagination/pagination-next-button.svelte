<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pagination as PaginationPrimitive } from 'bits-ui';
	import ChevronRight from 'lucide-svelte/icons/chevron-right';
	import { buttonVariants } from '@/components/ui/button/index.js';
	import { cn } from '@/components/ui/utils.js';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: PaginationPrimitive.NextButtonProps = $props();
</script>

{#snippet Fallback()}
	<span>Next</span>
	<ChevronRight class="size-4" />
{/snippet}

<PaginationPrimitive.NextButton
	bind:ref
	class={cn(
		buttonVariants({
			variant: 'ghost',
			class: 'gap-1 pr-2.5'
		}),
		className
	)}
	children={children || Fallback}
	{...restProps}
/>
