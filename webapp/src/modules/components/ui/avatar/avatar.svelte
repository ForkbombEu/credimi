<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Avatar as AvatarPrimitive } from "bits-ui";

	import { cn } from "@/components/ui/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: AvatarPrimitive.RootProps = $props();
</script>

<AvatarPrimitive.Root
	bind:ref
	class={cn("relative flex size-10 shrink-0 overflow-hidden rounded-full", className)}
	{...restProps}
/>
