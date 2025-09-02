<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WithElementRef } from 'bits-ui';
	import type { HTMLLiAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		children,
		...restProps
	}: WithElementRef<HTMLLiAttributes> = $props();
</script>

<li bind:this={ref} {...restProps}>
	{@render children?.()}
</li>
