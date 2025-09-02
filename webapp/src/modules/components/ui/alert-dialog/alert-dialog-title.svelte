<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { AlertDialog as AlertDialogPrimitive } from "bits-ui";

	import { cn } from "@/components/ui/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		level = 3,
		...restProps
	}: AlertDialogPrimitive.TitleProps = $props();
</script>

<AlertDialogPrimitive.Title
	bind:ref
	class={cn("text-lg font-semibold", className)}
	{level}
	{...restProps}
/>
