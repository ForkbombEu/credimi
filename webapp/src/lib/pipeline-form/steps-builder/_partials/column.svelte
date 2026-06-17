<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import T from '@/components/ui-custom/t.svelte';
	import * as Resizable from '@/components/ui/resizable/index.js';
	import { cn } from '@/components/ui/utils';

	type Props = {
		children?: Snippet;
		class?: string;
		contentClass?: string;
		disabled?: boolean;
		title: string;
		titleRight?: Snippet;
	};

	let {
		children,
		class: className,
		contentClass,
		disabled = false,
		title,
		titleRight
	}: Props = $props();

	const classes = $derived(
		cn('flex flex-col overflow-hidden rounded-lg border bg-white shadow-sm', className)
	);
</script>

<Resizable.Pane class={classes}>
	<div class="flex items-center justify-between border-b bg-slate-100 px-4 py-2">
		<T class="font-semibold">{title}</T>
		{@render titleRight?.()}
	</div>

	<div class={['relative flex grow flex-col overflow-y-scroll', contentClass]}>
		{#if disabled}
			<div class="absolute inset-0 z-10 bg-white/40" aria-hidden="true"></div>
		{/if}
		<div class={disabled ? 'opacity-60' : ''}>
			{@render children?.()}
		</div>
	</div>
</Resizable.Pane>
