<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		children?: Snippet;
		class?: string;
		title: string;
		titleRight?: Snippet;
	};

	let { children, class: className = '', title, titleRight }: Props = $props();
</script>

<div class={['flex flex-col overflow-hidden rounded-lg border bg-white shadow-sm', className]}>
	<div class="flex items-center justify-between border-b bg-slate-100 px-4 py-2">
		<T class="font-semibold">{title}</T>
		{@render titleRight?.()}
	</div>

	<div class="flex grow flex-col p-4">
		{@render children?.()}
	</div>
</div>
