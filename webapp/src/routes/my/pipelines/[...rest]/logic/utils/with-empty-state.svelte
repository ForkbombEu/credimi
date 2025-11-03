<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		items: T[];
		emptyText: string;
		item: Snippet<[{ item: T }]>;
		containerClass?: ClassValue;
	};

	let { items, emptyText, item: itemSnippet, containerClass }: Props = $props();
</script>

{#if items.length > 0}
	<div class={['space-y-2', containerClass]}>
		{#each items as item (item)}
			{@render itemSnippet?.({ item })}
		{/each}
	</div>
{:else}
	<div class="flex flex-col items-center justify-center rounded-md bg-gray-100 p-4">
		<T class="text-muted-foreground text-sm">{emptyText}</T>
	</div>
{/if}
