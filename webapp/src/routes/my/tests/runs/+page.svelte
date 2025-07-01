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
	import { WorkflowsTable } from '$lib/workflows';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n/index.js';
	import Button from '@/components/ui-custom/button.svelte';
	import { XIcon } from 'lucide-svelte';
	import { Separator } from '@/components/ui/separator/index.js';
	import { QrCode } from '@/qr';

	//

	let { data } = $props();
	const { executions } = $derived(data);

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
	{#if latestExecutions.length > 0}
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<T tag="h3">{m.Review_latest_check_runs()}</T>
				<Button
					variant="link"
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

				{#snippet rowRight({ workflow, Td })}
					<Td>
						<QrCode
							src={workflow.execution.runId}
							cellSize={40}
							class="aspect-square"
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
		<div
			class={[
				'space-y-4 transition-opacity duration-300',
				{ 'opacity-40 hover:opacity-100': latestExecutions.length > 0 }
			]}
		>
			<T tag="h3">{m.Checks_history()}</T>
			<WorkflowsTable workflows={oldExecutions} />
		</div>
	{/if}
</div>
