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
</script>

<div class="mx-auto max-w-7xl px-4 md:px-8">
	{#if legend}
		<div class="mb-4 flex flex-wrap items-center justify-end gap-4">{@render legend()}</div>
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
						<th
							class="sticky top-0 z-10 min-w-32 border-b bg-muted/60 px-3 py-3 text-center font-semibold"
						>
							<div class="inline-flex max-w-44 flex-col items-center gap-1">
								{#if matrix.column.tiered && column.tier === 'group'}
									<button
										type="button"
										class="inline-flex items-center gap-1 rounded-sm p-0.5 hover:bg-muted"
										aria-expanded={expandedColumnGroups.has(column.id)}
										aria-label={expandedColumnGroups.has(column.id)
											? m.interop_matrix_collapse_group()
											: m.interop_matrix_expand_group()}
										onclick={() => toggleColumnGroup(column.id)}
									>
										{#if expandedColumnGroups.has(column.id)}
											<ChevronDownIcon class="size-4 shrink-0" />
										{:else}
											<ChevronRightIcon class="size-4 shrink-0" />
										{/if}
										<span>{column.name}</span>
										<span class="text-xs font-normal text-muted-foreground">
											{groupChildCountLabel(column.child_count)}
										</span>
									</button>
								{:else}
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
								{/if}
							</div>
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each view.rows as row (row.id + row.tier)}
					<tr class="border-b last:border-b-0">
						<th
							class="sticky left-0 z-10 border-r bg-muted/40 px-3 py-3 text-left align-middle font-medium"
						>
							{#if matrix.row.tiered && row.tier === 'group'}
								<button
									type="button"
									class="inline-flex w-full items-center gap-2 rounded-sm text-left hover:bg-muted/60"
									aria-expanded={expandedRowGroups.has(row.id)}
									aria-label={expandedRowGroups.has(row.id)
										? m.interop_matrix_collapse_group()
										: m.interop_matrix_expand_group()}
									onclick={() => toggleRowGroup(row.id)}
								>
									{#if expandedRowGroups.has(row.id)}
										<ChevronDownIcon class="size-4 shrink-0" />
									{:else}
										<ChevronRightIcon class="size-4 shrink-0" />
									{/if}
									{#if row.avatar_url}
										<img
											src={row.avatar_url}
											alt={row.name}
											class="size-6 rounded-full object-cover"
											loading="lazy"
										/>
									{/if}
									<span class="min-w-0 flex-1">
										<span class="block">{row.name}</span>
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
							<td class="border-l p-2 align-top">
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
