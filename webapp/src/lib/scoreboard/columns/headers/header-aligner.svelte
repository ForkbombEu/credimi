<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	export type AlignProp = {
		align?: 'left' | 'center' | 'right';
	};
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	let { align = 'left', children }: AlignProp & { children: Snippet } = $props();
</script>

<div class="flex items-center gap-2 text-[1rem] text-primary">
	{@render children?.()}
</div>
