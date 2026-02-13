<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { SearchIcon, SparkleIcon, TestTubeIcon } from '@lucide/svelte';
	import { Pipeline } from '$lib';
	import TemporalI18nProvider from '$lib/temporal/temporal-i18n-provider.svelte';
	import { PolledResource } from '$lib/utils/state.svelte.js';
	import { WorkflowQrPoller, WorkflowsTable } from '$lib/workflows';
	import { queryParameters } from 'sveltekit-search-params';

	import Button from '@/components/ui-custom/button.svelte';
	import EmptyState from '@/components/ui-custom/emptyState.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as Tabs from '@/components/ui/tabs/index.js';
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../../+layout@.svelte';
	import { fetchWorkflows, isExtendedWorkflowStatus, TABS } from './_partials/index.js';
	import PaginationArrows from './_partials/pagination-arrows.svelte';

	//

	let { data } = $props();
	let { workflows: loadedWorkflows, pagination } = $derived(data);

	setDashboardNavbar({
		title: m.workflows()
	});

	const params = queryParameters({
		tab: {
			defaultValue: 'pipeline' as const,
			encode: (value) => value,
			decode: (value) => {
				if (value === 'other') return 'other';
				else return 'pipeline';
			}
		},
		status: {
			encode: (value) => value,
			decode: (value) => {
				if (isExtendedWorkflowStatus(value)) return value;
				else return undefined;
			}
		},
		limit: {
			encode: (value) => (value === undefined ? undefined : String(value)),
			decode: (value) => {
				const parsed = Number(value);
				return Number.isNaN(parsed) || value === null ? undefined : parsed;
			}
		},
		offset: {
			encode: (value) => (value === undefined ? undefined : String(value)),
			decode: (value) => {
				const parsed = Number(value);
				return Number.isNaN(parsed) || value === null ? undefined : parsed;
			}
		}
	});

	//

	const workflows = new PolledResource(
		async () => {
			const result = await fetchWorkflows(params.tab, {
				status: params.status,
				...pagination
			});
			if (result instanceof Error) {
				console.error(result);
				return [];
			}
			return result;
		},
		{
			initialValue: () => loadedWorkflows,
			intervalMs: 10000
		}
	);
</script>

<div class="grow space-y-8">
	<div class="flex items-center justify-between">
		<T tag="h3">{m.workflow_runs()}</T>
		<Tabs.Root bind:value={params.tab}>
			<Tabs.List class="gap-1 bg-secondary">
				{#each Object.entries(TABS) as [key, value] (key)}
					<Tabs.Trigger
						class="data-[state=inactive]:hover:cursor-pointer data-[state=inactive]:hover:bg-primary/10 "
						value={key}
					>
						{value}
					</Tabs.Trigger>
				{/each}
			</Tabs.List>
		</Tabs.Root>
	</div>

	{#if workflows.current?.length > 0}
		{#if params.tab === 'pipeline'}
			<PaginationArrows
				{pagination}
				onPrevious={() => {
					if (params.offset === undefined || params.offset === null) return;
					params.offset = params.offset - 1;
				}}
				onNext={() => {
					if (params.offset === undefined || params.offset === null) params.offset = 2;
					params.offset = params.offset + 1;
				}}
			/>
			<Pipeline.Workflows.Table workflows={workflows.current} />
		{:else}
			<WorkflowsTable workflows={workflows.current}>
				{#snippet header({ Th })}
					<Th>
						{m.QR_code()}
					</Th>
				{/snippet}
				{#snippet row({ workflow, Td })}
					<Td>
						{#if workflow.status === 'Running'}
							<WorkflowQrPoller
								workflowId={workflow.execution.workflowId}
								runId={workflow.execution.runId}
								containerClass="size-40"
							/>
						{:else}
							<span class="text-muted-foreground opacity-50">N/A</span>
						{/if}
					</Td>
				{/snippet}
			</WorkflowsTable>
		{/if}
	{/if}

	{#if workflows.current?.length === 0}
		{#if params.status}
			<EmptyState icon={SearchIcon} title={m.No_check_runs_with_this_status()}>
				{#snippet bottom()}
					<TemporalI18nProvider>
						<div class="pt-2">
							{#if params.status}
								<Pipeline.Workflows.StatusTag status={params.status} />
							{/if}
						</div>
					</TemporalI18nProvider>
				{/snippet}
			</EmptyState>
		{:else}
			<EmptyState icon={TestTubeIcon} title={m.No_test_runs_yet()} className="w-full">
				{#snippet bottom()}
					<div class="mt-4 space-y-3">
						<p class="text-sm text-muted-foreground">{m.Start_a_new_test_run()}</p>
						<div class="flex flex-col gap-2">
							<Button href="/my/pipelines">
								<SparkleIcon />
								{m.Execute_a_pipeline()}
							</Button>
							<Button href="/my/tests/new">
								{m.Start_a_manual_conformance_check()}
							</Button>
						</div>
					</div>
				{/snippet}
			</EmptyState>
		{/if}
	{/if}
</div>
