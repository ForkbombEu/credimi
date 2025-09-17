<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';

	import content from './content.yaml?raw';

	let width = $state(100);
	let baseWidth = 100;
	let t = 1;

	setInterval(() => {
		const cos = Math.cos(t);
		width = cos * cos * baseWidth + 100;
		t += 0.01;
	}, 10);
</script>

<div class="flex w-full items-center gap-2 p-4">
	<textarea style:--w="{width}px" class="block size-96 shrink-0 bg-red-500"></textarea>
	<CodeDisplay {content} language="yaml" class="max-h-80" />
</div>

<style>
	textarea {
		width: var(--w);
	}
</style>
