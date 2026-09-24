<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { createColumnHelper, getCoreRowModel } from '@tanstack/table-core';
	import {
		displayNameFromUid,
		displayStandardName,
		SuiteBrowse,
		type ConformanceSuiteRecord,
		type SuiteFacetKey
	} from '$lib/conformance';
	import { entities, type EntityData } from '$lib/global/entities';
	import EntityTag from '$lib/global/entity-tag.svelte';

	import SortHeaderPill from '@/components/ui-custom/sort-header-pill.svelte';
	import { createSvelteTable, FlexRender, renderComponent } from '@/components/ui/data-table';
	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import SuiteChecksCell from './_partials/suite-checks-cell.svelte';
	import TableNameCell from './_partials/table-name-cell.svelte';

	//

	type Props = {
		/** SSR suite rows (pipeline surface; used when sort/search/facets are default). */
		suites?: ConformanceSuiteRecord[];
		/** Debounced text search across suite display/identity fields. */
		search?: string;
	};

	let { suites: initialSuites = [], search = '' }: Props = $props();

	const browse = new SuiteBrowse({
		surface: 'pipeline',
		get initialSuites() {
			return initialSuites;
		},
		get search() {
			return search;
		}
	});

	const facetFields: { key: SuiteFacetKey; label: string }[] = [
		{ key: 'standard', label: m.Standard() },
		{ key: 'component', label: m.Component() },
		{ key: 'provider', label: m.Provider() }
	];

	const selectClass =
		'border-input bg-background flex h-9 min-w-[8rem] rounded-md border px-3 py-1 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]';

	const columnHelper = createColumnHelper<ConformanceSuiteRecord>();

	const columns = $derived([
		columnHelper.accessor('suite_name', {
			id: 'suite',
			header: m.Suite(),
			enableSorting: true,
			cell: ({ row }) =>
				renderComponent(TableNameCell, {
					name: suiteLabel(row.original),
					href: `/hub/conformance-checks/${row.original.path_prefix}`,
					logo: row.original.suite_logo || undefined
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

	/** Prefer authored metadata `name` (suite_name); fall back to suite uid. */
	function suiteLabel(suite: ConformanceSuiteRecord): string {
		const name = suite.suite_name?.trim();
		if (name) return name;
		return displayNameFromUid(suite.suite);
	}

	function entityForComponent(component: string): EntityData | undefined {
		switch (component.trim()) {
			case 'wallet':
				return entities.wallets;
			case 'issuer':
				return entities.credential_issuers;
			case 'verifier':
				return entities.verifiers;
			default:
				return undefined;
		}
	}

	function versionLabel(version: string): string {
		return version.trim() || '—';
	}

	function facetOptionLabel(key: SuiteFacetKey, value: string): string {
		switch (key) {
			case 'standard':
				return displayStandardName(value);
			case 'component': {
				const entity = entityForComponent(value);
				return entity?.labels.singular ?? value;
			}
			case 'provider':
				return value;
		}
	}
</script>

<div class="space-y-4">
	<div class="flex flex-wrap items-end gap-3 px-4 pt-4">
		{#each facetFields as { key, label } (key)}
			<div class="flex flex-col gap-1">
				<label class="text-muted-foreground text-xs" for={`facet-${key}`}>{label}</label>
				<select id={`facet-${key}`} class={selectClass} bind:value={browse.filters[key]}>
					<option value="">{m.All()}</option>
					{#each browse.facetOptions[key] as value (value)}
						<option {value}>{facetOptionLabel(key, value)}</option>
					{/each}
				</select>
			</div>
		{/each}

		{#if browse.hasActiveFilters}
			<button
				type="button"
				class="text-primary text-sm underline-offset-4 hover:underline"
				onclick={browse.clearFilters}
			>
				{m.Clear_filters()}
			</button>
		{/if}
	</div>

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
</div>
