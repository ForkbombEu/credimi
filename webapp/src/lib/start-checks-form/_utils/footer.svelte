<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		right?: Snippet;
		left?: Snippet;
	};

	const { right, left }: Props = $props();
</script>

<div class="bg-background sticky bottom-0 rounded-md border p-3 px-6 backdrop-blur-lg">
	<div class="mx-auto flex max-w-screen-xl items-center justify-between gap-8">
		<div class="flex shrink-0 text-sm">
			{@render left?.()}
		</div>

		<div class="flex items-start gap-4">
			{@render right?.()}
		</div>
	</div>
</div>
