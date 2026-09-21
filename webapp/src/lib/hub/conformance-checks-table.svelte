<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import {
		listAll,
		listChecks,
		type CatalogFacets,
		type ConformanceCheckRecord,
		type StandardsWithTestSuites
	} from '$lib/conformance';

	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import ContentWrapper from './_partials/content-wrapper.svelte';
	import ChildrenCell from './_partials/table-children-cell.svelte';
	import TableNameCell from './_partials/table-name-cell.svelte';

	//

	type Props = {
		standardsWithTestSuites?: StandardsWithTestSuites;
	};

	let { standardsWithTestSuites = [] }: Props = $props();

	type FacetField = keyof CatalogFacets;

	type FacetOptions = Record<FacetField, string[]>;

	const facetFields: { key: FacetField; label: string }[] = [
		{ key: 'protocol', label: 'Protocol' },
		{ key: 'sut', label: 'SUT' },
		{ key: 'role', label: 'Role' },
		{ key: 'provider', label: 'Provider' }
	];

	const selectClass =
		'border-input bg-background flex h-9 min-w-[8rem] rounded-md border px-3 py-1 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]';

	let filters = $state<Record<FacetField, string>>({
		protocol: '',
		sut: '',
		role: '',
		provider: ''
	});

	let facetOptions = $state<FacetOptions>({
		protocol: [],
		sut: [],
		role: [],
		provider: []
	});

	let displayedStandards = $state<StandardsWithTestSuites>([]);
	let isLoading = $state(false);
	let catalogRequestId = 0;

	const activeFacets = $derived.by((): CatalogFacets => {
		const facets: CatalogFacets = {};
		for (const { key } of facetFields) {
			const value = filters[key];
			if (value) facets[key] = value;
		}
		return facets;
	});

	const hasActiveFilters = $derived(Object.keys(activeFacets).length > 0);

	const rows = $derived(
		displayedStandards.flatMap((standard) =>
			standard.versions.flatMap((version) =>
				version.suites.map((suite) => ({ standard, version, suite }))
			)
		)
	);

	function collectFacetOptions(records: ConformanceCheckRecord[]) {
		const distinct = (field: FacetField) =>
			[...new Set(records.map((record) => record[field]).filter((value) => value.length > 0))].sort();

		facetOptions = {
			protocol: distinct('protocol'),
			sut: distinct('sut'),
			role: distinct('role'),
			provider: distinct('provider')
		};
	}

	function clearFilters() {
		filters.protocol = '';
		filters.sut = '';
		filters.role = '';
		filters.provider = '';
	}

	$effect(() => {
		listChecks({ surface: 'pipeline' }).match({
			Rejected: (reason) => {
				console.error(reason);
			},
			Resolved: collectFacetOptions
		});
	});

	$effect(() => {
		const facets = activeFacets;
		const hasFilters = Object.keys(facets).length > 0;
		const requestId = ++catalogRequestId;

		if (!hasFilters) {
			if (standardsWithTestSuites.length > 0) {
				displayedStandards = standardsWithTestSuites;
				isLoading = false;
				return;
			}

			isLoading = true;
			listAll({ surface: 'pipeline' }).match({
				Rejected: (reason) => {
					if (requestId !== catalogRequestId) return;
					console.error(reason);
					isLoading = false;
				},
				Resolved: (standards) => {
					if (requestId !== catalogRequestId) return;
					displayedStandards = standards;
					isLoading = false;
				}
			});
			return;
		}

		isLoading = true;
		listAll({ surface: 'pipeline', facets }).match({
			Rejected: (reason) => {
				if (requestId !== catalogRequestId) return;
				console.error(reason);
				isLoading = false;
			},
			Resolved: (standards) => {
				if (requestId !== catalogRequestId) return;
				displayedStandards = standards;
				isLoading = false;
			}
		});
	});
</script>

<div class="space-y-4 px-4 pb-4">
	<div class="flex flex-wrap items-end gap-3">
		{#each facetFields as { key, label } (key)}
			<div class="flex flex-col gap-1">
				<label class="text-xs text-muted-foreground" for={`facet-${key}`}>{label}</label>
				<select id={`facet-${key}`} class={selectClass} bind:value={filters[key]}>
					<option value="">All</option>
					{#each facetOptions[key] as value (value)}
						<option {value}>{value}</option>
					{/each}
				</select>
			</div>
		{/each}

		{#if hasActiveFilters}
			<button
				type="button"
				class="text-sm text-primary underline-offset-4 hover:underline"
				onclick={clearFilters}
			>
				Clear filters
			</button>
		{/if}
	</div>

	<div class:opacity-60={isLoading}>
		<Table.Table>
			<Table.Header>
				<Table.Head class="px-4">{m.Standard()}</Table.Head>
				<Table.Head class="px-4">{m.Version()}</Table.Head>
				<Table.Head class="px-4">{m.Suite()}</Table.Head>
				<Table.Head class="px-4">{m.Checks()}</Table.Head>
			</Table.Header>
			<Table.Body>
				{#each rows as { standard, version, suite } (standard.uid + version.uid + suite.uid)}
					<Table.Row>
						<Table.Cell class="px-4 align-top">
							<TableNameCell
								name={standard.name}
								href={`/hub/conformance-checks/${standard.uid}/${version.uid}/${suite.uid}`}
								logo={suite.logo}
							/>
						</Table.Cell>
						<Table.Cell class="px-4 align-top">
							<ContentWrapper>
								{version.name}
							</ContentWrapper>
						</Table.Cell>
						<Table.Cell class="px-4 align-top">
							<ContentWrapper>
								{suite.name}
							</ContentWrapper>
						</Table.Cell>
						<Table.Cell class="px-4">
							{#if standard.uid === 'fcaf'}
								<span class="text-xs text-muted-foreground">
									{suite.paths.length} {m.Tests()}
								</span>
							{:else if suite.files.length > 0}
								<ChildrenCell
									items={suite.paths.map((p, i) => ({
										title: suite.files[i].replace('.yaml', ''),
										href: `/hub/conformance-checks/${p}`
									}))}
								/>
							{/if}
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Table>
	</div>
</div>
