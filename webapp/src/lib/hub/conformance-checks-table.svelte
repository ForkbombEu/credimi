<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { SearchIcon } from '@lucide/svelte';
	import { createColumnHelper, getCoreRowModel } from '@tanstack/table-core';
	import {
		displayStandardName,
		entityForComponent,
		suiteHubHref,
		suiteLogo,
		suiteSubtitle,
		suiteTitle,
		SuiteBrowse,
		type ConformanceSuiteRecord
	} from '$lib/conformance';
	import EntityTag from '$lib/global/entity-tag.svelte';

	import EmptyState from '@/components/ui-custom/emptyState.svelte';
	import SortHeaderPill from '@/components/ui-custom/sort-header-pill.svelte';
	import { createSvelteTable, FlexRender, renderComponent } from '@/components/ui/data-table';
	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import SuiteChecksCell from './_partials/suite-checks-cell.svelte';
	import TableNameCell from './_partials/table-name-cell.svelte';

	//

	type Props = {
		/** Shared hub suite browse state (filters live in the page header). */
		browse: SuiteBrowse;
		/** Debounced text search (empty-state + check-cell highlight). */
		search?: string;
	};

	let { browse, search = '' }: Props = $props();

	const columnHelper = createColumnHelper<ConformanceSuiteRecord>();

	const columns = $derived([
		columnHelper.accessor('suite_name', {
			id: 'suite',
			header: m.Suite(),
			enableSorting: true,
			cell: ({ row }) =>
				renderComponent(TableNameCell, {
					name: suiteTitle(row.original),
					subtitle: suiteSubtitle(row.original),
					href: suiteHubHref(row.original),
					logo: suiteLogo(row.original)
				})
		}),
		columnHelper.accessor('standard', {
			id: 'standard',
			header: m.Standard(),
			enableSorting: true,
			cell: ({ row }) => displayStandardName(row.original.standard)
		}),
		columnHelper.accessor('component', {
			id: 'component',
			header: m.Component(),
			enableSorting: true,
			cell: ({ row }) => {
				const data = entityForComponent(row.original.component);
				if (!data) return '—';
				return renderComponent(EntityTag, { data });
			}
		}),
		columnHelper.accessor('version', {
			id: 'version',
			header: m.Version(),
			enableSorting: false,
			cell: ({ row }) => versionLabel(row.original.version)
		}),
		columnHelper.accessor('check_count', {
			id: 'checks',
			header: m.Checks(),
			enableSorting: false,
			cell: ({ row }) =>
				renderComponent(SuiteChecksCell, { suite: row.original, search })
		})
	]);

	const table = createSvelteTable({
		get data() {
			return browse.displayedSuites;
		},
		get columns() {
			return columns;
		},
		getCoreRowModel: getCoreRowModel(),
		getRowId: (row) => row.id,
		manualSorting: true,
		state: {
			get sorting() {
				return browse.sorting;
			}
		},
		onSortingChange: (updater) => {
			browse.sorting =
				typeof updater === 'function' ? updater(browse.sorting) : updater;
		}
	});

	function versionLabel(version: string): string {
		return version.trim() || '—';
	}
</script>

{#if search.trim() && browse.displayedSuites.length === 0 && !browse.isLoading}
	<EmptyState
		title={m.No_records_found()}
		icon={SearchIcon}
		className="rounded-none border-0"
	/>
{:else}
	<div class:opacity-60={browse.isLoading}>
		<Table.Table>
			<Table.Header>
				{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
					<Table.Row>
						{#each headerGroup.headers as header (header.id)}
							<Table.Head class="px-4">
								{#if !header.isPlaceholder}
									{#if header.column.getCanSort()}
										<button
											type="button"
											class="group relative flex items-center gap-1 text-left hover:cursor-pointer"
											onclick={header.column.getToggleSortingHandler()}
										>
											<FlexRender
												content={header.column.columnDef.header}
												context={header.getContext()}
											/>
											<SortHeaderPill {header} {table} />
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
				{#each table.getRowModel().rows as row (row.id)}
					<Table.Row>
						{#each row.getVisibleCells() as cell (cell.id)}
							<Table.Cell class="px-4 align-top">
								<div class="flex min-h-[41px] items-center">
									<FlexRender
										content={cell.column.columnDef.cell}
										context={cell.getContext()}
									/>
								</div>
							</Table.Cell>
						{/each}
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Table>
	</div>
{/if}
