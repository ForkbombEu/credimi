<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowDownIcon, ArrowUpDownIcon, ArrowUpIcon } from '@lucide/svelte';

	import { FlexRender } from '@/components/ui/data-table/index.js';
	import * as Pagination from '@/components/ui/pagination/index.js';
	import * as Table from '@/components/ui/table/index.js';

	import type { ScoreboardTable } from './table.svelte.ts';

	//

	let { scoreboard }: { scoreboard: ScoreboardTable } = $props();
</script>

<div class="space-y-4">
	<div class="overflow-hidden rounded-md bg-background">
		<Table.Root>
			<Table.Header>
				{#each scoreboard.table.getHeaderGroups() as headerGroup (headerGroup.id)}
					<Table.Row class="bg-[#d1c8f3] hover:bg-[#d1c8f3]">
						{#each headerGroup.headers as header (header.id)}
							<Table.Head colspan={header.colSpan}>
								{#if !header.isPlaceholder}
									{#if header.column.getCanSort()}
										{@const sorted = header.column.getIsSorted()}
										<button
											type="button"
											class="group relative hover:cursor-pointer"
											onclick={header.column.getToggleSortingHandler()}
										>
											<FlexRender
												content={header.column.columnDef.header}
												context={header.getContext()}
											/>
											<div
												class={[
													'absolute -top-1 right-0 translate-x-2 rounded-full bg-primary p-1 text-primary-foreground opacity-0 transition-opacity',
													'duration-150',
													'group-hover:bg-blue-700 group-hover:opacity-100',
													sorted ? 'opacity-100' : 'opacity-0'
												]}
											>
												{#if sorted === 'asc'}
													<ArrowUpIcon class="size-3" />
												{:else if sorted === 'desc'}
													<ArrowDownIcon class="size-3" />
												{:else}
													<ArrowUpDownIcon class="size-3 opacity-50" />
												{/if}
											</div>
										</button>
									{:else}
										<FlexRender
											content={header.column.columnDef.header}
											context={header.getContext()}
										/>
									{/if}
								{/if}
							</Table.Head>
						{/each}
					</Table.Row>
				{/each}
			</Table.Header>
			<Table.Body>
				{#each scoreboard.table.getRowModel().rows as row (row.id)}
					<Table.Row data-state={row.getIsSelected() && 'selected'}>
						{#each row.getVisibleCells() as cell (cell.id)}
							<Table.Cell class="align-top">
								<FlexRender
									content={cell.column.columnDef.cell}
									context={cell.getContext()}
								/>
							</Table.Cell>
						{/each}
					</Table.Row>
				{:else}
					<Table.Row>
						<Table.Cell
							colspan={scoreboard.table.getAllColumns().length}
							class="h-24 text-center"
						>
							No results.
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
	</div>

	{#if scoreboard.pageCount > 1}
		<div class="flex items-center justify-center space-x-2 py-4">
			<Pagination.Root
				count={scoreboard.totalItems}
				perPage={scoreboard.pageSize}
				bind:page={scoreboard.currentPage}
			>
				{#snippet children({ pages, currentPage })}
					<Pagination.Content>
						<Pagination.Item>
							<Pagination.Previous />
						</Pagination.Item>
						{#each pages as page (page.key)}
							{#if page.type === 'ellipsis'}
								<Pagination.Item>
									<Pagination.Ellipsis />
								</Pagination.Item>
							{:else}
								<Pagination.Item>
									<Pagination.Link {page} isActive={currentPage === page.value}>
										{page.value}
									</Pagination.Link>
								</Pagination.Item>
							{/if}
						{/each}
						<Pagination.Item>
							<Pagination.Next />
						</Pagination.Item>
					</Pagination.Content>
				{/snippet}
			</Pagination.Root>
		</div>
	{/if}
</div>
