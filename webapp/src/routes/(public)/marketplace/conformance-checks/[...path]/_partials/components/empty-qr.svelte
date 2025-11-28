<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	let { children } = $props();
</script>

<div
	class={[
		'aspect-square size-60 shrink-0 overflow-hidden rounded-md border',
		'flex items-center justify-center',
		'text-muted-foreground bg-gray-50',
		'p-3 text-center text-sm'
	]}
>
	{@render children?.()}
</div>
