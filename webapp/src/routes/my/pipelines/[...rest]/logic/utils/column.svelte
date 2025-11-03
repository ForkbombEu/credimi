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
		title?: string;
	};

	let { children, class: className = '', title }: Props = $props();
</script>

<div class={['flex flex-col gap-4 rounded-lg border bg-white p-4 shadow-sm lg:p-6', className]}>
	{#if title}
		<T class="border-b pb-1 font-semibold">{title}</T>
	{/if}
	{@render children?.()}
</div>
