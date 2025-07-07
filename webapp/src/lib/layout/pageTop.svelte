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
	<div class={['mx-auto max-w-screen-xl space-y-8 px-8 py-12 pt-20', contentClass]}>
		{@render children()}
	</div>
</div>
