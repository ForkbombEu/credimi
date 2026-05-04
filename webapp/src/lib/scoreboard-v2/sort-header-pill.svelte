<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Header, Table } from '@tanstack/table-core';

	import { ArrowDownIcon, ArrowUpDownIcon, ArrowUpIcon } from '@lucide/svelte';

	import type { ScoreboardRow } from './types';

	type Props = {
		header: Header<ScoreboardRow, unknown>;
		table: Table<ScoreboardRow>;
	};

	let { header, table }: Props = $props();

	const sorted = $derived(header.column.getIsSorted());
	const sortIndex = $derived(header.column.getSortIndex());
	const multiSort = $derived(table.getState().sorting.length > 1);
	const showSortRank = $derived(multiSort && Boolean(sorted) && sortIndex >= 0);
</script>

<div
	class={[
		'absolute -top-1 right-0 flex translate-x-2 items-center gap-0.5 rounded-full bg-primary px-1 py-1 text-primary-foreground opacity-0 transition-opacity',
		'duration-150',
		'group-hover:bg-blue-700 group-hover:opacity-100',
		sorted ? 'opacity-100' : 'opacity-0'
	]}
>
	{#if sorted === 'asc'}
		<ArrowUpIcon class="size-3 shrink-0" />
		{#if showSortRank}
			<span class="text-[10px] font-semibold leading-none tabular-nums">{sortIndex + 1}</span>
		{/if}
	{:else if sorted === 'desc'}
		<ArrowDownIcon class="size-3 shrink-0" />
		{#if showSortRank}
			<span class="text-[10px] font-semibold leading-none tabular-nums">{sortIndex + 1}</span>
		{/if}
	{:else}
		<ArrowUpDownIcon class="size-3 opacity-50" />
	{/if}
</div>
