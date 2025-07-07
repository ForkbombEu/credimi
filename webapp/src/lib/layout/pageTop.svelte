<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		children: Snippet;
		contentClass?: string;
	};

	const { children, contentClass = '' }: Props = $props();
</script>

<div class="border-y-2 border-y-primary bg-[url(/hero.png)] bg-cover bg-right">
	<div class="mx-auto max-w-screen-xl space-y-8 px-8 py-12 pt-20 {contentClass}">
		{@render children()}
	</div>
</div>
