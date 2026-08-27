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

	const listClass = $derived.by(() => {
		if (layout === 'links-only' && items.length === 1) {
			return ['flex h-[30px] items-center', className];
		}

		const base: Record<Layout, string> = {
			compact: 'flex flex-wrap items-center gap-4',
			full: 'flex flex-col gap-2',
			'avatar-only': 'flex flex-col gap-0',
			'links-only': 'flex flex-col gap-0'
		};

		return [
			base[layout],
			layout !== 'compact' && (align === 'end' ? 'items-end' : 'items-start'),
			className
		];
	});
</script>

<div class={listClass}>
	{#each items as item (item.key)}
		<EntityItem {item} {layout} />
	{:else}
		<Na />
	{/each}
</div>
