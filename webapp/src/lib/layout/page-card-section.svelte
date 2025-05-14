<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PlainCard from '@/components/ui-custom/plainCard.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import type { Snippet } from 'svelte';

	type Props = {
		title?: string;
		description?: string;
		children?: Snippet;
		class?: string;
	};

	let { title, description, children, class: className = '' }: Props = $props();
</script>

<PlainCard class="space-y-6 !p-6 md:space-y-8 md:!p-8 {className}">
	{#if title || description}
		<div class="space-y-1 border-b pb-2">
			{#if title}
				<T tag="h4">
					{title}
				</T>
			{/if}
			{#if description}
				<T class="text-muted-foreground ">
					{description}
				</T>
			{/if}
		</div>
	{/if}

	{@render children?.()}
</PlainCard>
