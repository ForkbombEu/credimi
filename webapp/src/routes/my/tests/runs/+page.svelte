<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import {
		LatestCheckRunsStorage,
		type StartCheckResultWithMeta
	} from '$lib/start-checks-form/_utils';
	import { browser } from '$app/environment';
	import { Array } from 'effect';
	import { ensureArray } from '@/utils/other';
	import { WorkflowQrPoller, WorkflowsTable } from '$lib/workflows';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n/index.js';
	import Button from '@/components/ui-custom/button.svelte';
	import { SearchIcon, SparkleIcon, TestTube2, XIcon } from 'lucide-svelte';
	import { Separator } from '@/components/ui/separator/index.js';
	import WorkflowStatusSelect from '$lib/workflows/workflow-status-select.svelte';
	import EmptyState from '@/components/ui-custom/emptyState.svelte';
	import { Badge } from '@/components/ui/badge/index.js';
	import { setWorkflowStatusesInUrl } from './utils.js';

	//

	let { data } = $props();
	const { executions, selectedStatuses } = $derived(data);

	let latestCheckRuns: StartCheckResultWithMeta[] = $state(
		browser ? ensureArray(LatestCheckRunsStorage.get()) : []
	);
	const latestRunIds = $derived(latestCheckRuns.map((run) => run.WorkflowRunId));

	const latestExecutions = $derived(
		executions.filter((exec) => latestRunIds.includes(exec.execution.runId))
	);

	const oldExecutions = $derived(Array.difference(executions, latestExecutions));
</script>

<div class="space-y-8">
	<div class="bg-background flex flex-wrap items-center gap-4 rounded-lg border p-4">
		<p>Filter runs by status</p>
		<WorkflowStatusSelect value={selectedStatuses} onValueChange={setWorkflowStatusesInUrl} />
	</div>

	{#if latestExecutions.length > 0}
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

			<WorkflowsTable workflows={latestExecutions}>
				{#snippet headerRight({ Th })}
					<Th>
						{m.QR_code()}
					</Th>
				{/snippet}

				{#snippet rowRight({ workflow, Td, workflowMemo })}
					<Td>
						<WorkflowQrPoller
							workflowId={workflow.execution.workflowId}
							runId={workflow.execution.runId}
							containerClass="size-32"
						/>
					</Td>
				{/snippet}
			</WorkflowsTable>
		</div>
	{/if}

	{#if oldExecutions.length !== 0 && latestExecutions.length !== 0}
		<Separator />
	{/if}

	{#if oldExecutions.length > 0}
		<div class="space-y-4">
			<T tag="h3">{m.Checks_history()}</T>
			<WorkflowsTable workflows={oldExecutions} />
		</div>
	{/if}

	{#if oldExecutions.length === 0 && latestExecutions.length === 0}
		{#if selectedStatuses.length === 0}
			<EmptyState
				icon={TestTube2}
				title={m.No_check_runs_yet()}
				description={m.Start_a_new_check_run_to_see_it_here()}
			>
				{#snippet bottom()}
					<Button href="/my/tests/new" variant="outline" class="text-primary mt-4">
						<SparkleIcon />
						{m.Start_a_new_check()}
						<Badge
							variant="outline"
							class="border-primary text-primary !hover:no-underline text-xs"
						>
							{m.Beta()}
						</Badge>
					</Button>
				{/snippet}
			</EmptyState>
		{:else}
			<EmptyState icon={SearchIcon} title={m.No_check_runs_with_this_status()} />
		{/if}
	{/if}
</div>
