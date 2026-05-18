<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		children: Snippet;
	};

	let { children }: Props = $props();
</script>

<div
	class="flex flex-col items-center justify-center rounded-lg bg-primary/5 p-10 text-center text-primary/20"
>
	{@render children()}
</div>
