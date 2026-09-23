<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowDownIcon, ArrowUpDownIcon, ArrowUpIcon } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import {
		createColumnHelper,
		getCoreRowModel,
		type SortingState
	} from '@tanstack/table-core';
	import {
		displayNameFromUid,
		displayStandardName,
		listSuites,
		type ConformanceSuiteRecord
	} from '$lib/conformance';
	import { entities, type EntityData } from '$lib/global/entities';
	import EntityTag from '$lib/global/entity-tag.svelte';

	import {
		createSvelteTable,
		FlexRender,
		renderComponent
	} from '@/components/ui/data-table';
	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import SuiteChecksCell from './_partials/suite-checks-cell.svelte';
	import TableNameCell from './_partials/table-name-cell.svelte';

	//

	type Props = {
		/** SSR suite rows (pipeline surface, no facets). */
		suites?: ConformanceSuiteRecord[];
	};

	let { suites: initialSuites = [] }: Props = $props();

	/** Stable API default (includes version tie-break even though Version is not UI-sortable). */
	const DEFAULT_SORT = 'standard,component,version,suite';

	const DEFAULT_SORTING: SortingState = [
		{ id: 'standard', desc: false },
		{ id: 'component', desc: false },
		{ id: 'suite', desc: false }
	];

	/** Column id → PocketBase sort field. Suite sorts by authored name. */
	const SORT_FIELDS: Record<string, string> = {
		standard: 'standard',
		component: 'component',
		suite: 'suite_name'
	};

	let sorting = $state<SortingState>([...DEFAULT_SORTING]);

	const sortString = $derived(buildSortString(sorting));

	const useSSR = $derived(sortString === DEFAULT_SORT && initialSuites.length > 0);

	const catalogQuery = createQuery(() => {
		const sort = sortString;

		return {
			queryKey: ['conformance-suites', 'pipeline', 'hub', sort] as const,
			enabled: !useSSR,
			queryFn: async () => {
				const result = await listSuites({
					surface: 'pipeline',
					sort
				});
				if (result.isErr) throw result.error;
				return result.value;
			}
		};
	});

	const displayedSuites = $derived.by((): ConformanceSuiteRecord[] => {
		if (useSSR) return initialSuites;
		return catalogQuery.data ?? [];
	});

	const isLoading = $derived(catalogQuery.isFetching);

	const columnHelper = createColumnHelper<ConformanceSuiteRecord>();

	const columns = [
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
			cell: ({ row }) => renderComponent(SuiteChecksCell, { suite: row.original })
		})
	];

	const table = createSvelteTable({
		get data() {
			return displayedSuites;
		},
		columns,
		getCoreRowModel: getCoreRowModel(),
		getRowId: (row) => row.id,
		manualSorting: true,
		state: {
			get sorting() {
				return sorting;
			}
		},
		onSortingChange: (updater) => {
			sorting = typeof updater === 'function' ? updater(sorting) : updater;
		}
	});

	function isDefaultSorting(state: SortingState): boolean {
		if (state.length !== DEFAULT_SORTING.length) return false;
		return state.every(
			(entry, index) =>
				entry.id === DEFAULT_SORTING[index].id && entry.desc === DEFAULT_SORTING[index].desc
		);
	}

	function buildSortString(state: SortingState): string {
		if (state.length === 0 || isDefaultSorting(state)) return DEFAULT_SORT;
		return state
			.map(({ id, desc }) => {
				const field = SORT_FIELDS[id];
				if (!field) return null;
				return desc ? `-${field}` : field;
			})
			.filter((part): part is string => Boolean(part))
			.join(',');
	}

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

	function sortIcon(sorted: false | 'asc' | 'desc') {
		if (sorted === 'asc') return ArrowUpIcon;
		if (sorted === 'desc') return ArrowDownIcon;
		return ArrowUpDownIcon;
	}
</script>

<div class="space-y-4 px-4 pb-4">
	<div class:opacity-60={isLoading}>
		<Table.Table>
			<Table.Header>
				{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
					<Table.Row>
						{#each headerGroup.headers as header (header.id)}
							<Table.Head class="px-4">
								{#if !header.isPlaceholder}
									{#if header.column.getCanSort()}
										{@const SortIcon = sortIcon(header.column.getIsSorted())}
										<button
											type="button"
											class="inline-flex items-center gap-1 text-left hover:cursor-pointer"
											onclick={header.column.getToggleSortingHandler()}
										>
											<FlexRender
												content={header.column.columnDef.header}
												context={header.getContext()}
											/>
											<SortIcon class="size-3.5 shrink-0 text-muted-foreground" />
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
