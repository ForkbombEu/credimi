<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Select as SelectPrimitive, type WithoutChildrenOrChild } from "bits-ui";
	import ChevronDown from "lucide-svelte/icons/chevron-down";

	import { cn } from "@/components/ui/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: WithoutChildrenOrChild<SelectPrimitive.ScrollDownButtonProps> = $props();
</script>

<SelectPrimitive.ScrollDownButton
	bind:ref
	class={cn("flex cursor-default items-center justify-center py-1", className)}
	{...restProps}
>
	<ChevronDown class="size-4" />
</SelectPrimitive.ScrollDownButton>
