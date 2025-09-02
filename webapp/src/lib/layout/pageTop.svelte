<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'clsx';
	import type { Snippet } from 'svelte';

	type Props = {
		children: Snippet;
		contentClass?: string;
		containerClass?: ClassValue;
		hideTopBorder?: boolean;
	};

	const {
		children,
		contentClass = '',
		containerClass = '',
		hideTopBorder = false
	}: Props = $props();
</script>

<div
	class={[
		'border-b-2 border-primary bg-[url(/hero.png)] bg-cover bg-right',
		{ 'border-t-2': !hideTopBorder },
		containerClass
	]}
>
	<div
		class={[
			'mx-auto max-w-screen-xl space-y-6 px-4 py-6 pt-8 sm:space-y-8 md:px-8 md:py-12 md:pt-20',
			contentClass
		]}
	>
		{@render children()}
	</div>
</div>
