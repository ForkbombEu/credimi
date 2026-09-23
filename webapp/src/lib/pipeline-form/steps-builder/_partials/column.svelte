<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { PaneHandle } from '$pipeline-form/steps-builder/pane-layout.js';
	import type { Snippet } from 'svelte';
	import type { Attachment } from 'svelte/attachments';

	import T from '@/components/ui-custom/t.svelte';
	import * as Resizable from '@/components/ui/resizable/index.js';
	import { cn } from '@/components/ui/utils';

	type Props = {
		children?: Snippet;
		class?: string;
		contentClass?: string;
		defaultSize?: number;
		disabled?: boolean;
		minSize?: number;
		order?: number;
		pane?: PaneHandle | null;
		scrollAttach?: Attachment;
		scrollContainer?: HTMLElement | null;
		title: string;
		titleRight?: Snippet;
	};

	let {
		children,
		class: className,
		contentClass,
		defaultSize,
		disabled = false,
		minSize,
		order,
		pane = $bindable<PaneHandle | null>(null),
		scrollAttach,
		scrollContainer = $bindable<HTMLElement | null>(null),
		title,
		titleRight
	}: Props = $props();

	const classes = $derived(
		cn('flex flex-col overflow-hidden rounded-lg border bg-white shadow-sm', className)
	);
</script>

<Resizable.Pane bind:this={pane} class={classes} {defaultSize} {minSize} {order}>
	<div class="flex min-w-0 items-center justify-between gap-2 border-b bg-slate-100 px-4 py-2">
		<T class="shrink-0 font-semibold">{title}</T>
		<div class="min-w-0 shrink">
			{@render titleRight?.()}
		</div>
	</div>

	<div
		bind:this={scrollContainer}
		{@attach scrollAttach}
		class={['relative flex min-h-0 grow flex-col overflow-y-scroll', contentClass]}
	>
		{#if disabled}
			<div class="absolute inset-0 z-10 bg-white/40" aria-hidden="true"></div>
		{/if}
		<div class={['flex min-h-0 grow flex-col', disabled && 'opacity-60']}>
			{@render children?.()}
		</div>
	</div>
</Resizable.Pane>
