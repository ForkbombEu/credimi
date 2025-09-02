<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Dialog as SheetPrimitive } from 'bits-ui';

	import { cn } from '@/components/ui/utils.js';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: SheetPrimitive.DescriptionProps = $props();
</script>

<SheetPrimitive.Description
	bind:ref
	class={cn('text-sm text-muted-foreground', className)}
	{...restProps}
/>
