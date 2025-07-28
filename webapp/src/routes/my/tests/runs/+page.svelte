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
	import { INVALIDATE_KEY, setWorkflowStatusesInUrl } from './utils.js';
	import { onMount } from 'svelte';
	import { invalidate } from '$app/navigation';

	//

	let { data } = $props();
	let { workflows, selectedStatuses } = $derived(data);

	let latestCheckRuns: StartCheckResultWithMeta[] = $state(
		browser ? ensureArray(LatestCheckRunsStorage.get()) : []
	);
	const latestRunIds = $derived(latestCheckRuns.map((run) => run.WorkflowRunID));

	const latestWorkflows = $derived(workflows.filter((w) => latestRunIds.includes(w.runId)));
	const oldWorkflows = $derived(Array.difference(workflows, latestWorkflows));

	//

	onMount(() => {
		const interval = setInterval(async () => {
			invalidate(INVALIDATE_KEY);
		}, 5000);

		return () => {
			clearInterval(interval);
		};
	});
</script>

<div class="space-y-8">
	<div class="bg-background flex flex-wrap items-center gap-4 rounded-lg border p-4">
		<p>{m.Filter_runs_by_status()}</p>
		<WorkflowStatusSelect value={selectedStatuses} onValueChange={setWorkflowStatusesInUrl} />
	</div>

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

			<WorkflowsTable workflows={latestWorkflows} separateLogs>
				{#snippet headerRight({ Th })}
					<Th>
						{m.QR_code()}
					</Th>
				{/snippet}

				{#snippet rowRight({ workflow, Td })}
					<Td>
						{#if workflow.status === 'Running'}
							<WorkflowQrPoller
								workflowId={workflow.id}
								runId={workflow.runId}
								containerClass="size-32"
							/>
						{/if}
					</Td>
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
			<WorkflowsTable workflows={oldWorkflows} separateLogs />
		</div>
	{/if}

	{#if oldWorkflows.length === 0 && latestWorkflows.length === 0}
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
