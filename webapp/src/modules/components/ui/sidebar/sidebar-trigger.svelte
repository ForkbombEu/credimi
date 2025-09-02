<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from "svelte";

	import PanelLeft from "lucide-svelte/icons/panel-left";

	import { Button } from "@/components/ui/button/index.js";
	import { cn } from "@/components/ui/utils.js";

	import { useSidebar } from "./context.svelte.js";

	let {
		ref = $bindable(null),
		class: className,
		onclick,
		...restProps
	}: ComponentProps<typeof Button> & {
		onclick?: (e: MouseEvent) => void;
	} = $props();

	const sidebar = useSidebar();
</script>

<Button
	type="button"
	onclick={(e) => {
		onclick?.(e);
		sidebar.toggle();
	}}
	data-sidebar="trigger"
	variant="ghost"
	size="icon"
	class={cn("h-7 w-7", className)}
	{...restProps}
>
	<PanelLeft />
	<span class="sr-only">Toggle Sidebar</span>
</Button>
