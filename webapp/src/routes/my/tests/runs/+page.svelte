<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { SearchIcon, SparkleIcon, TestTubeIcon } from '@lucide/svelte';
	import TemporalI18nProvider from '$lib/temporal/temporal-i18n-provider.svelte';
	import { PolledResource } from '$lib/utils/state.svelte.js';
	import { fetchWorkflows, WorkflowsTable } from '$lib/workflows';
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
		title: m.Test_runs()
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
		other: m.Conformance_and_custom_checks(),
		pipeline: m.Pipelines()
	} as const;

	type SelectedTab = keyof typeof tabs;
	let selectedTab = $state<SelectedTab>('other');

	const selectedWorkflows = $derived(
		selectedTab === 'pipeline' ? pipelineWorkflows : otherWorkflows
	);

	// let latestCheckRuns: StartCheckResultWithMeta[] = $state([]);
	// if (browser) latestCheckRuns = ensureArray(LatestCheckRunsStorage.get());

	// const latestRunIds = $derived(latestCheckRuns.map((run) => run.workflowRunId));
	// const latestWorkflows = $derived(
	// 	workflows.filter((w) => latestRunIds.includes(w.execution.runId))
	// );
	// const oldWorkflows = $derived(Array.difference(workflows, latestWorkflows));

	// onMount(() => {
	// 	const interval = setInterval(async () => {
	// 		const newWorkflows = await fetchWorkflows({ status: selectedStatus });
	// 		if (newWorkflows instanceof Error) warn(newWorkflows);
	// 		else workflows = newWorkflows;
	// 	}, 5000);

	// 	return () => {
	// 		clearInterval(interval);
	// 	};
	// });
</script>

<div class="grow space-y-8">
	<div class="flex items-center justify-between">
		<T tag="h3">{m.Checks_history()}</T>
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

	<!-- {#if latestWorkflows.length > 0}
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<T tag="h3">{m.Review_latest_check_runs()}</T>
				<Button
					variant="outline"
					size="sm"
					onclick={() => {
						latestCheckRuns = [];
						LatestCheckRunsStorage.remove();
					}}
				>
					<XIcon />
					<span>
						{m.Clear_list()}
					</span>
				</Button>
			</div>

			<WorkflowsTable workflows={latestWorkflows}>
				{#snippet nameRight({ workflow })}
					{@const status = toWorkflowStatusReadable(workflow.status)}
					{#if status === 'Running'}
						<WorkflowQrPoller
							workflowId={workflow.execution.workflowId}
							runId={workflow.execution.runId}
							containerClass="size-40"
						/>
					{/if}
				{/snippet}
			</WorkflowsTable>
		</div>
	{/if}

	{#if oldWorkflows.length !== 0 && latestWorkflows.length !== 0}
		<Separator />
	{/if} -->

	{#if selectedWorkflows.length > 0}
		<WorkflowsTable workflows={selectedWorkflows} hideResults={selectedTab === 'other'} />
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
