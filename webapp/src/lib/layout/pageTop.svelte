<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		children: Snippet;
		contentClass?: string;
		containerClass?: string;
	};

	const { children, contentClass = '', containerClass = '' }: Props = $props();
</script>

<div class={['border-y-primary border-y-2 bg-[url(/hero.png)] bg-cover bg-right', containerClass]}>
	<div class={['mx-auto max-w-screen-xl px-4 py-6 space-y-6 pt-8 md:px-8 md:py-12 sm:space-y-8 md:pt-20', contentClass]}>
		{@render children()}
	</div>
</div>
