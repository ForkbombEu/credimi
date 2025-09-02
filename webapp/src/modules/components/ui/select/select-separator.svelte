<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Separator as SeparatorPrimitive } from "bits-ui";

	import { Separator } from "@/components/ui/separator/index.js";
	import { cn } from "@/components/ui/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: SeparatorPrimitive.RootProps = $props();
</script>

<Separator bind:ref class={cn("bg-muted -mx-1 my-1 h-px", className)} {...restProps} />
