<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Standard } from '$lib/conformance/types';
	import type { Snippet } from 'svelte';

	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import { SvelteSet } from 'svelte/reactivity';

	import { m } from '@/i18n';

	import type { InteropMatrixResponse } from './types';

	import MatrixCell from './matrix-cell.svelte';
	import { buildVisibleMatrix, cellKey } from './to-view-matrix';

	type Props = {
		matrix: InteropMatrixResponse;
		standards: readonly Standard[];
		expandedRowGroups?: Set<string>;
		expandedColumnGroups?: Set<string>;
		legend?: Snippet;
	};

	let {
		matrix,
		standards,
		expandedRowGroups = $bindable(new SvelteSet<string>()),
		expandedColumnGroups = $bindable(new SvelteSet<string>()),
		legend
	}: Props = $props();

	const view = $derived(
		buildVisibleMatrix(matrix, {
			standards,
			expandedRowGroups,
			expandedColumnGroups
		})
	);

	function toggleRowGroup(groupId: string) {
		const next = new SvelteSet(expandedRowGroups);
		if (next.has(groupId)) {
			next.delete(groupId);
		} else {
			next.add(groupId);
		}
		expandedRowGroups = next;
	}

	function toggleColumnGroup(groupId: string) {
		const next = new SvelteSet(expandedColumnGroups);
		if (next.has(groupId)) {
			next.delete(groupId);
		} else {
			next.add(groupId);
		}
		expandedColumnGroups = next;
	}

	function groupChildCountLabel(count: number | undefined): string {
		return count === undefined ? '' : `(${count})`;
	}

	function isGroupExpanded(groupId: string, expanded: ReadonlySet<string>): boolean {
		return expanded.has(groupId);
	}
</script>

{#snippet groupChevron(
	groupId: string,
	expanded: Set<string>,
	onToggle: (id: string) => void
)}
	<button
		type="button"
		class="inline-flex shrink-0 items-center justify-center rounded-sm p-0.5 hover:bg-muted"
		aria-expanded={isGroupExpanded(groupId, expanded)}
		aria-label={isGroupExpanded(groupId, expanded)
			? m.interop_matrix_collapse_group()
			: m.interop_matrix_expand_group()}
		onclick={() => onToggle(groupId)}
	>
		{#if isGroupExpanded(groupId, expanded)}
			<ChevronDownIcon class="size-4 text-foreground" aria-hidden="true" />
		{:else}
			<ChevronRightIcon class="size-4 text-foreground" aria-hidden="true" />
		{/if}
	</button>
{/snippet}

{#snippet groupToggleButton(
	groupId: string,
	expanded: Set<string>,
	onToggle: (id: string) => void,
	label: string,
	childCount?: number
)}
	<button
		type="button"
		class="inline-flex items-center gap-1.5 rounded-sm p-0.5 text-left hover:bg-muted"
		aria-expanded={isGroupExpanded(groupId, expanded)}
		aria-label={isGroupExpanded(groupId, expanded)
			? m.interop_matrix_collapse_group()
			: m.interop_matrix_expand_group()}
		onclick={() => onToggle(groupId)}
	>
		{#if isGroupExpanded(groupId, expanded)}
			<ChevronDownIcon class="size-4 shrink-0 text-foreground" aria-hidden="true" />
		{:else}
			<ChevronRightIcon class="size-4 shrink-0 text-foreground" aria-hidden="true" />
		{/if}
		<span>{label}</span>
		{#if childCount !== undefined}
			<span class="text-xs font-normal text-muted-foreground">
				{groupChildCountLabel(childCount)}
			</span>
		{/if}
	</button>
{/snippet}

<div class="mx-auto max-w-7xl px-4 md:px-8">
	{#if legend}
		<div class="mb-4 flex flex-wrap items-center justify-end gap-4">
			{@render legend()}
		</div>
	{/if}

	<div class="overflow-x-auto rounded-lg border bg-background shadow-sm">
		<table class="w-full min-w-[640px] border-collapse text-sm">
			<thead>
				<tr>
					<th
						class="sticky left-0 z-20 min-w-40 border-r border-b bg-muted/80 px-3 py-3 text-left text-xs font-semibold tracking-wide text-muted-foreground uppercase"
					>
						{view.cornerLabel}
					</th>
					{#each view.columns as column (column.id + column.tier)}
						{@const columnGroupExpanded =
							column.tier === 'group' && expandedColumnGroups.has(column.id)}
						<th
							class="sticky top-0 z-10 border-b bg-muted/60 text-center font-semibold {columnGroupExpanded
								? 'w-9 min-w-9 max-w-9 px-0 py-1'
								: 'min-w-32 px-3 py-3'}"
						>
							{#if column.tier === 'group' && columnGroupExpanded}
								<div class="flex min-h-7 items-center justify-center">
									{@render groupChevron(
										column.id,
										expandedColumnGroups,
										toggleColumnGroup
									)}
								</div>
							{:else if column.tier === 'group'}
								<div class="inline-flex max-w-44 flex-col items-center gap-1">
									{@render groupToggleButton(
										column.id,
										expandedColumnGroups,
										toggleColumnGroup,
										column.name,
										column.child_count
									)}
								</div>
							{:else}
								<div class="inline-flex max-w-44 flex-col items-center gap-1">
									<a
										class="inline-flex flex-col items-center gap-1 hover:underline"
										href={column.href}
									>
										{#if column.avatar_url}
											<img
												src={column.avatar_url}
												alt={column.name}
												class="size-6 rounded-full object-cover"
												loading="lazy"
											/>
										{/if}
										<span>{column.name}</span>
										{#if column.displaySubtitle}
											<span class="text-xs font-normal text-muted-foreground">
												{column.displaySubtitle}
											</span>
										{/if}
									</a>
								</div>
							{/if}
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each view.rows as row (row.id + row.tier)}
					{@const rowGroupExpanded =
						row.tier === 'group' && expandedRowGroups.has(row.id)}
					<tr
						class="border-b last:border-b-0 {rowGroupExpanded ? 'h-8' : ''}"
					>
						<th
							class="sticky left-0 z-10 border-r bg-muted/40 text-left align-middle font-medium {rowGroupExpanded
								? 'px-1 py-0.5'
								: row.nested
									? 'px-3 py-3 ps-8'
									: 'px-3 py-3'}"
						>
							{#if row.tier === 'group' && rowGroupExpanded}
								<div class="flex min-h-7 items-center">
									{@render groupChevron(
										row.id,
										expandedRowGroups,
										toggleRowGroup
									)}
								</div>
							{:else if row.tier === 'group'}
								<button
									type="button"
									class="inline-flex w-full items-center gap-2 rounded-sm text-left hover:bg-muted/60"
									aria-expanded={false}
									aria-label={m.interop_matrix_expand_group()}
									onclick={() => toggleRowGroup(row.id)}
								>
									<ChevronRightIcon
										class="size-4 shrink-0 text-foreground"
										aria-hidden="true"
									/>
									{#if row.avatar_url}
										<img
											src={row.avatar_url}
											alt={row.name}
											class="size-6 shrink-0 rounded-full object-cover"
											loading="lazy"
										/>
									{/if}
									<span class="min-w-0 flex-1">
										<span class="block font-medium">{row.name}</span>
										<span class="text-xs font-normal text-muted-foreground">
											{groupChildCountLabel(row.child_count)}
										</span>
									</span>
								</button>
							{:else}
								<a
									class="inline-flex items-center gap-2 hover:underline"
									href={row.href}
								>
									{#if row.avatar_url}
										<img
											src={row.avatar_url}
											alt={row.name}
											class="size-6 rounded-full object-cover"
											loading="lazy"
										/>
									{/if}
									<span>{row.name}</span>
								</a>
								{#if row.displaySubtitle}
									<span
										class="mt-0.5 block text-xs font-normal text-muted-foreground"
									>
										{row.displaySubtitle}
									</span>
								{/if}
							{/if}
						</th>
						{#each view.columns as column (column.id + column.tier)}
							{@const columnGroupExpanded =
								column.tier === 'group' && expandedColumnGroups.has(column.id)}
							<td
								class="border-l align-top {rowGroupExpanded || columnGroupExpanded
									? 'p-0.5'
									: 'p-2'}"
							>
								<MatrixCell
									cell={view.cells.get(
										cellKey(row.tier, row.id, column.tier, column.id)
									)}
								/>
							</td>
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
