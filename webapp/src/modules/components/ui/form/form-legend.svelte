<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WithoutChild } from 'bits-ui';

	import * as FormPrimitive from 'formsnap';

	import { cn } from '@/components/ui/utils.js';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: WithoutChild<FormPrimitive.LegendProps> = $props();
</script>

<FormPrimitive.Legend
	bind:ref
	class={cn('text-sm font-medium leading-none data-[fs-error]:text-destructive', className)}
	{...restProps}
/>
