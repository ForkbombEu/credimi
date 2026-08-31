<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	import * as conformanceChecks from './columns/conformance-checks.svelte';
	import * as issuers from './columns/issuers.svelte';
	import * as lastExecution from './columns/last-execution.svelte';
	import * as name from './columns/name.svelte';
	import * as runners from './columns/runners.svelte';
	import * as verifiers from './columns/verifiers.svelte';
	import * as videoScreenshot from './columns/video-screenshot.svelte';
	import * as wallets from './columns/wallets.svelte';

	const COLUMN_TOGGLES = [
		{ id: name.column.id, label: m.Pipeline() },
		{ id: videoScreenshot.column.id, label: m.Evidence() },
		{ id: issuers.column.id, label: m.Issuance() },
		{ id: wallets.column.id, label: m.Wallets() },
		{ id: verifiers.column.id, label: m.Presentations() },
		{ id: conformanceChecks.column.id, label: m.Conformance_Checks() },
		{ id: runners.column.id, label: m.Runners() },
		{ id: lastExecution.column.id, label: m.scoreboard_last_run() }
	];

	import { CheckIcon, Columns3Icon } from '@lucide/svelte';

	const BAND_FILTERS = [
		{ value: 'all', label: m.All(), filter: undefined },
		{ value: 'stable', label: m.scoreboard_band_stable(), filter: 'success_rate >= 80' },
		{
			value: 'flaky',
			label: m.scoreboard_band_flaky(),
			filter: 'success_rate >= 60 && success_rate < 80'
		},
		{
			value: 'failing',
			label: m.scoreboard_band_failing(),
			filter: 'success_rate >= 30 && success_rate < 60'
		},
		{ value: 'broken', label: m.scoreboard_band_broken(), filter: 'success_rate < 30' }
	];
</script>

<script lang="ts">
	import { ChevronDownIcon } from '@lucide/svelte';

	import HorizontalScrollArea from '@/components/ui-custom/horizontal-scroll-area.svelte';
	import Button from '@/components/ui/button/button.svelte';
	import { FlexRender } from '@/components/ui/data-table/index.js';
	import * as DropdownMenu from '@/components/ui/dropdown-menu/index.js';
	import Input from '@/components/ui/input/input.svelte';
	import * as Pagination from '@/components/ui/pagination/index.js';
	import * as Select from '@/components/ui/select/index.js';
	import * as Table from '@/components/ui/table/index.js';
	import { pb } from '@/pocketbase';

	import type { ScoreboardTable } from './table.svelte.ts';

	import HeaderContextProvider from './columns/headers/header-context-provider.svelte';
	import RowDetails from './row-details.svelte';
	import SortHeaderPill from './sort-header-pill.svelte';

	//

	let { scoreboard }: { scoreboard: ScoreboardTable } = $props();

	let expandedRowId = $state<string | undefined>(undefined);

	function toggleRow(id: string) {
		expandedRowId = expandedRowId === id ? undefined : id;
	}

	// Click anywhere on the row toggles expansion; links and inner buttons keep their own behavior.
	function onRowClick(event: MouseEvent, id: string) {
		if ((event.target as HTMLElement).closest('a, button')) return;
		toggleRow(id);
	}

	// Rows change on pagination/sort: collapse so details never leak across rows.
	$effect(() => {
		void scoreboard.table.getRowModel().rows;
		expandedRowId = undefined;
	});

	let searchInput = $state('');
	let bandFilter = $state('all');
	let searchDebounce: ReturnType<typeof setTimeout> | undefined;

	const scrollRefresh = $derived({
		page: scoreboard.currentPage,
		rows: scoreboard.table.getRowModel().rows.length
	});

	function applyFilters() {
		const parts: string[] = [];
		const query = searchInput.trim();
		if (query) {
			parts.push(pb.filter('pipeline.name ~ {:q}', { q: query }));
		}
		const band = BAND_FILTERS.find((option) => option.value === bandFilter);
		if (band?.filter) {
			parts.push(band.filter);
		}
		scoreboard.setFilter(parts.length > 0 ? parts.join(' && ') : undefined);
	}

	function onSearchInput() {
		clearTimeout(searchDebounce);
		searchDebounce = setTimeout(applyFilters, 300);
	}
</script>

<div class="space-y-4">
	<div class="flex flex-wrap items-center gap-2">
		<Input
			type="search"
			placeholder={m.Search()}
			class="w-56"
			bind:value={searchInput}
			oninput={onSearchInput}
		/>
		<Select.Root
			type="single"
			value={bandFilter}
			onValueChange={(value) => {
				bandFilter = value;
				applyFilters();
			}}
		>
			<Select.Trigger class="w-40">
				{BAND_FILTERS.find((option) => option.value === bandFilter)?.label}
			</Select.Trigger>
			<Select.Content>
				{#each BAND_FILTERS as option (option.value)}
					<Select.Item value={option.value} label={option.label}
						>{option.label}</Select.Item
					>
				{/each}
			</Select.Content>
		</Select.Root>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Button {...props} variant="outline" size="sm">
						<Columns3Icon class="size-4" />
						{m.Columns()}
					</Button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content class="w-56">
				<DropdownMenu.Label>{m.Columns()}</DropdownMenu.Label>
				{#each COLUMN_TOGGLES as column (column.id)}
					{@const visible = scoreboard.columnVisibility[column.id] !== false}
					<DropdownMenu.Item
						onSelect={(event) => {
							event.preventDefault();
							scoreboard.toggleColumn(column.id, !visible);
						}}
					>
						<CheckIcon class={['size-4', !visible && 'text-transparent']} />
						{column.label}
					</DropdownMenu.Item>
				{/each}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</div>

	<HorizontalScrollArea class="overflow-hidden rounded-md bg-background" refresh={scrollRefresh}>
		<table class="w-max min-w-full caption-bottom text-sm">
			<Table.Header>
				{#each scoreboard.table.getHeaderGroups() as headerGroup (headerGroup.id)}
					<Table.Row class="bg-secondary">
						<Table.Head class="w-10"></Table.Head>
						{#each headerGroup.headers as header (header.id)}
							{#if header.column.getIsVisible()}
								<Table.Head colspan={header.colSpan}>
									<HeaderContextProvider {header} table={scoreboard.table}>
										{#if !header.isPlaceholder}
											{#if header.column.getCanSort()}
												<button
													type="button"
													class="group relative flex items-center gap-1 hover:cursor-pointer"
													onclick={header.column.getToggleSortingHandler()}
												>
													<FlexRender
														content={header.column.columnDef.header}
														context={header.getContext()}
													/>
													{#if !header.column.columnDef.meta?.manualPillPositioning}
														<SortHeaderPill
															{header}
															table={scoreboard.table}
														/>
													{/if}
												</button>
											{:else}
												<FlexRender
													content={header.column.columnDef.header}
													context={header.getContext()}
												/>
											{/if}
										{/if}
									</HeaderContextProvider>
								</Table.Head>
							{/if}
						{/each}
					</Table.Row>
				{/each}
			</Table.Header>
			<Table.Body>
				{#each scoreboard.table.getRowModel().rows as row (row.id)}
					{@const isExpanded = expandedRowId === row.id}
					<Table.Row
						data-state={row.getIsSelected() && 'selected'}
						class="cursor-pointer hover:bg-secondary data-[state=selected]:bg-secondary {isExpanded
							? 'bg-secondary'
							: ''}"
						onclick={(event) => onRowClick(event, row.id)}
					>
						<Table.Cell class="align-top">
							<button
								type="button"
								class="flex size-7 items-center justify-center rounded-full text-muted-foreground hover:bg-secondary hover:text-foreground focus-visible:outline-2"
								aria-expanded={isExpanded}
								aria-label={isExpanded ? m.Collapse() : m.Expand()}
								onclick={() => toggleRow(row.id)}
							>
								<ChevronDownIcon
									class="size-4 transition-transform duration-150 {isExpanded
										? 'rotate-180'
										: ''}"
								/>
							</button>
						</Table.Cell>
						{#each row.getVisibleCells() as cell (cell.id)}
							<Table.Cell class="align-top whitespace-normal">
								<FlexRender
									content={cell.column.columnDef.cell}
									context={cell.getContext()}
								/>
							</Table.Cell>
						{/each}
					</Table.Row>
					{#if isExpanded}
						<Table.Row class="bg-secondary hover:bg-secondary">
							<Table.Cell colspan={row.getVisibleCells().length + 1} class="p-0!">
								<RowDetails record={row.original} />
							</Table.Cell>
						</Table.Row>
					{/if}
				{:else}
					<Table.Row>
						<Table.Cell
							colspan={scoreboard.table.getVisibleLeafColumns().length + 1}
							class="h-24 text-center"
						>
							No results.
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</table>
	</HorizontalScrollArea>

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
