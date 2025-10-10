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
	class="text-primary/40 flex flex-col items-center justify-center rounded-lg border p-10 text-center"
>
	{@render children()}
</div>
