<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import T from '@/components/ui-custom/t.svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	type Props = HTMLAttributes<HTMLDivElement> & {
		children: Snippet;
		title?: string;
	};

	const { children, class: className, title, ...restProps }: Props = $props();
</script>

<div class={['bg-background space-y-6 rounded-md p-6 shadow-sm', className]} {...restProps}>
	{#if title}
		<T tag="h4" class="overflow-auto border-b pb-1">{title}</T>
	{/if}
	{@render children?.()}
</div>
