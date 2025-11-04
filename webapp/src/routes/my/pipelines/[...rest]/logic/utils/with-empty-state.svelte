<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import T from '@/components/ui-custom/t.svelte';
	import { ScrollArea } from '@/components/ui/scroll-area';

	type Props = {
		items: T[];
		emptyText: string;
		item: Snippet<[{ item: T }]>;
		containerClass?: ClassValue;
	};

	let { items, emptyText, item: itemSnippet, containerClass }: Props = $props();
</script>

{#if items.length > 0}
	<ScrollArea class={['grow [&>div>div]:space-y-2 [&>div>div]:p-4', containerClass]}>
		{#each items as item (item)}
			{@render itemSnippet?.({ item })}
		{/each}
	</ScrollArea>
{:else}
	<div class="p-4">
		<div class="flex flex-col items-center justify-center rounded-md bg-slate-100 p-4">
			<T class="text-muted-foreground text-sm">{emptyText}</T>
		</div>
	</div>
{/if}
