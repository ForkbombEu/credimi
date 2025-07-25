<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		children: Snippet;
		title: string;
	};

	let { children, title }: Props = $props();
</script>

<div class="space-y-2">
	<h2 class=" text-xl font-medium">{title}</h2>

	<div class="border-subtle bg-white p-2">
		{@render children?.()}
	</div>
</div>

<style>
	.border-subtle {
		border: 1px solid rgb(174 190 217);
	}
</style>
