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
		title: string;
		titleRight?: Snippet;
	};

	let { children, class: className, contentClass, title, titleRight }: Props = $props();

	const classes = cn(
		'flex flex-col overflow-hidden rounded-lg border bg-white shadow-sm',
		className
	);
</script>

<Resizable.Pane class={classes}>
	<div class="flex items-center justify-between border-b bg-slate-100 px-4 py-2">
		<T class="font-semibold">{title}</T>
		{@render titleRight?.()}
	</div>

	<div class={['flex grow flex-col overflow-hidden', contentClass]}>
		{@render children?.()}
	</div>
</Resizable.Pane>
