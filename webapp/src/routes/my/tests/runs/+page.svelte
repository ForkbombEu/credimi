<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { SearchIcon, SparkleIcon, TestTubeIcon } from '@lucide/svelte';
	import { Pipeline } from '$lib';
	import TemporalI18nProvider from '$lib/temporal/temporal-i18n-provider.svelte';
	import { PolledResource } from '$lib/utils/state.svelte.js';
	import { fetchWorkflows, WorkflowQrPoller, WorkflowsTable } from '$lib/workflows';
	import { Array } from 'effect';

	import Button from '@/components/ui-custom/button.svelte';
	import EmptyState from '@/components/ui-custom/emptyState.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as Tabs from '@/components/ui/tabs/index.js';
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../../+layout@.svelte';

	//

	let { data } = $props();
	let { workflows: loadedWorkflows, selectedStatus } = $derived(data);

	setDashboardNavbar({
		title: m.workflows()
	});

	//

	const workflows = new PolledResource(
		async () => {
			const result = await fetchWorkflows({ status: selectedStatus });
			if (result instanceof Error) {
				console.error(result);
				return [];
			}
			return result;
		},
		{
			initialValue: () => loadedWorkflows,
			intervalMs: 3000
		}
	);

	const pipelineWorkflows = $derived.by(() => {
		const list = workflows.current?.filter((w) => w.type.name === 'Dynamic Pipeline Workflow');
		return list ?? [];
	});

	const otherWorkflows = $derived(Array.difference(workflows.current, pipelineWorkflows));

	//

	const tabs = {
		pipeline: m.Pipelines(),
		other: m.Conformance_and_custom_checks()
	} as const;

	type SelectedTab = keyof typeof tabs;
	let selectedTab = $state<SelectedTab>('pipeline');

	const selectedWorkflows = $derived(
		selectedTab === 'pipeline' ? pipelineWorkflows : otherWorkflows
	);
</script>

<div class="grow space-y-8">
	<div class="flex items-center justify-between">
		<T tag="h3">{m.workflow_runs()}</T>
		<Tabs.Root bind:value={selectedTab}>
			<Tabs.List class="gap-1 bg-secondary">
				{#each Object.entries(tabs) as [key, value] (key)}
					<Tabs.Trigger
						class="data-[state=inactive]:hover:cursor-pointer data-[state=inactive]:hover:bg-primary/10 "
						value={key}>{value}</Tabs.Trigger
					>
				{/each}
			</Tabs.List>
		</Tabs.Root>
	</div>

	{#if selectedWorkflows.length > 0}
		{#if selectedTab === 'pipeline'}
			<Pipeline.Workflows.Table workflows={pipelineWorkflows} />
		{:else}
			<WorkflowsTable workflows={otherWorkflows}>
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

	{#if selectedWorkflows.length === 0}
		{#if selectedStatus}
			<EmptyState icon={SearchIcon} title={m.No_check_runs_with_this_status()}>
				{#snippet bottom()}
					<TemporalI18nProvider>
						<div class="pt-2">
							<WorkflowStatus status={selectedStatus} />
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
