<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		left?: Snippet;
		center?: Snippet;
		right?: Snippet;
		class?: string;
	}

	let { left, center, right, class: className = '' }: Props = $props();
</script>

<div class="shrink-0 border-b {className}">
	<div class="mx-auto flex max-w-site items-center justify-between p-2">
		<div class="flex h-full shrink-0 flex-row items-center">
			{@render left?.()}
		</div>
		<div class="flex items-center justify-center gap-2">
			{@render center?.()}
		</div>
		<div class="flex shrink-0 items-center">
			{@render right?.()}
		</div>
	</div>
</div>
