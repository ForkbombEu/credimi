<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		children: Snippet;
		class?: string;
		contentClass?: string;
	};

	const { children, class: className = '', contentClass = '' }: Props = $props();
</script>

<div class={className}>
	<div class="mx-auto max-w-screen-xl px-4 md:px-6 lg:px-8 py-6 md:py-12 {contentClass}">
		{@render children()}
	</div>
</div>
