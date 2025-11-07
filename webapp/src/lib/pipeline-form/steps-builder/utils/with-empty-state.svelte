<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import { ScrollArea } from '@/components/ui/scroll-area';

	import EmptyState from './empty-state.svelte';

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
	<EmptyState text={emptyText} />
{/if}
