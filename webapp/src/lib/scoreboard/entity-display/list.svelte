<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import type { Align, Item, Layout } from './types';

	import EntityItem from './item.svelte';
	import Na from './na.svelte';

	//

	type Props = {
		items: Item[];
		layout: Layout;
		align?: Align;
		class?: ClassValue;
	};

	let { items, layout, align = 'start', class: className }: Props = $props();

	const listClass = $derived(
		layout === 'compact'
			? ['flex flex-wrap items-center gap-4', className]
			: layout === 'links-only' && items.length === 1
				? ['flex h-[30px] items-center', className]
				: [
						'flex flex-col gap-1',
						align === 'end' ? 'items-end' : 'items-start',
						layout === 'full' ? 'gap-2' : '',
						className
					]
	);
</script>

<div class={listClass}>
	{#each items as item (item.key)}
		<EntityItem {item} {layout} />
	{:else}
		<Na />
	{/each}
</div>
