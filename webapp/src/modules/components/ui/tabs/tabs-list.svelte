<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Tabs as TabsPrimitive } from "bits-ui";

	import { cn } from "@/components/ui/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: TabsPrimitive.ListProps = $props();
</script>

<TabsPrimitive.List
	bind:ref
	class={cn(
		"bg-muted text-muted-foreground inline-flex h-10 items-center justify-center rounded-md p-1",
		className
	)}
	{...restProps}
/>
