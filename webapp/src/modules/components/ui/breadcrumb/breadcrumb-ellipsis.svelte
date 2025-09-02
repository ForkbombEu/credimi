<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WithElementRef, WithoutChildren } from "bits-ui";
	import type { HTMLAttributes } from "svelte/elements";

	import Ellipsis from "lucide-svelte/icons/ellipsis";

	import { cn } from "@/components/ui/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: WithoutChildren<WithElementRef<HTMLAttributes<HTMLSpanElement>>> = $props();
</script>

<span
	bind:this={ref}
	role="presentation"
	aria-hidden="true"
	class={cn("flex size-9 items-center justify-center", className)}
	{...restProps}
>
	<Ellipsis class="size-4" />
	<span class="sr-only">More</span>
</span>
