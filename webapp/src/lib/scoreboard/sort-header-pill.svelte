<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Header, Table } from '@tanstack/table-core';
	import type { ClassValue } from 'svelte/elements';

	import { ArrowDownIcon, ArrowUpDownIcon, ArrowUpIcon } from '@lucide/svelte';

	import type { ScoreboardRow } from './types';

	//

	type Props = {
		header: Header<ScoreboardRow, unknown>;
		table: Table<ScoreboardRow>;
		class?: ClassValue;
	};

	let { header, table, class: className }: Props = $props();

	const sorted = $derived(header.column.getIsSorted());
	const sortIndex = $derived(header.column.getSortIndex());
	const multiSort = $derived(table.getState().sorting.length > 1);
	const showSortRank = $derived(multiSort && Boolean(sorted) && sortIndex >= 0);
</script>

<div
	class={[
		'flex items-center gap-0.5 rounded-full bg-primary p-1 text-primary-foreground',
		'transition-opacity duration-150',
		'group-hover:bg-blue-700 group-hover:opacity-100',
		sorted ? 'opacity-100' : 'opacity-0',
		className
	]}
>
	{#if sorted === 'asc'}
		<ArrowUpIcon class="size-3 shrink-0" />
		{#if showSortRank}
			<span class="text-[10px] leading-none font-semibold tabular-nums">{sortIndex + 1}</span>
		{/if}
	{:else if sorted === 'desc'}
		<ArrowDownIcon class="size-3 shrink-0" />
		{#if showSortRank}
			<span class="text-[10px] leading-none font-semibold tabular-nums">{sortIndex + 1}</span>
		{/if}
	{:else}
		<ArrowUpDownIcon class="size-3 opacity-50" />
	{/if}
</div>
