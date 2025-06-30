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

{#if latestExecutions.length > 0}
	<WorkflowsTable workflows={latestExecutions} />
{/if}

{#if oldExecutions.length > 0}
	<WorkflowsTable workflows={oldExecutions} />
{/if}
