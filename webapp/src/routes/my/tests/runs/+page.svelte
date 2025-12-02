<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { toWorkflowStatusReadable, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { browser } from '$app/environment';
	import {
		LatestCheckRunsStorage,
		type StartCheckResultWithMeta
	} from '$lib/start-checks-form/_utils';
	import TemporalI18nProvider from '$lib/temporal/temporal-i18n-provider.svelte';
	import { fetchWorkflows, WorkflowQrPoller, WorkflowsTable } from '$lib/workflows';
	import { Array } from 'effect';
	import { SearchIcon, SparkleIcon, TestTube2, XIcon } from 'lucide-svelte';
	import { onMount } from 'svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import EmptyState from '@/components/ui-custom/emptyState.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Separator } from '@/components/ui/separator/index.js';
	import { m } from '@/i18n';
	import { ensureArray, warn } from '@/utils/other';

	import { setDashboardNavbar } from '../../+layout@.svelte';

	//

	let { data } = $props();
	let { workflows, selectedStatus } = $derived(data);

	setDashboardNavbar({
		title: m.Test_runs()
	});

	//

	let latestCheckRuns: StartCheckResultWithMeta[] = $state([]);
	if (browser) latestCheckRuns = ensureArray(LatestCheckRunsStorage.get());

	const latestRunIds = $derived(latestCheckRuns.map((run) => run.workflowRunId));
	const latestWorkflows = $derived(
		workflows.filter((w) => latestRunIds.includes(w.execution.runId))
	);
	const oldWorkflows = $derived(Array.difference(workflows, latestWorkflows));

	onMount(() => {
		const interval = setInterval(async () => {
			const newWorkflows = await fetchWorkflows({ status: selectedStatus });
			if (newWorkflows instanceof Error) warn(newWorkflows);
			else workflows = newWorkflows;
		}, 5000);

		return () => {
			clearInterval(interval);
		};
	});
</script>

<div class="grow space-y-8">
	{#if latestWorkflows.length > 0}
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
							containerClass="size-32"
						/>
					{/if}
				{/snippet}
			</WorkflowsTable>
		</div>
	{/if}

	{#if oldWorkflows.length !== 0 && latestWorkflows.length !== 0}
		<Separator />
	{/if}

	{#if oldWorkflows.length > 0}
		<div class="space-y-4">
			<T tag="h3">{m.Checks_history()}</T>
			<WorkflowsTable workflows={oldWorkflows} />
		</div>
	{/if}

	{#if oldWorkflows.length === 0 && latestWorkflows.length === 0}
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
			<EmptyState icon={TestTube2} title={m.No_test_runs_yet()} className="w-full">
				{#snippet bottom()}
					<div class="mt-4 space-y-3">
						<p class="text-muted-foreground text-sm">{m.Start_a_new_test_run()}</p>
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
