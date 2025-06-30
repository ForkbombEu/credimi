<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import A from '@/components/ui-custom/a.svelte';
	import * as Table from '@/components/ui/table/index.js';
	import { toWorkflowStatusReadable, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import TemporalI18nProvider from './[workflow_id]/[run_id]/components/temporal-i18n-provider.svelte';
	import { toUserTimezone } from '@/utils/toUserTimezone';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { LatestCheckRunsStorage } from '$lib/start-checks-form/_utils';
	import { browser } from '$app/environment';
	import { m } from '@/i18n';
	import Button from '@/components/ui-custom/button.svelte';
	import { ArrowRightIcon, InfoIcon, X } from 'lucide-svelte';
	import T from '@/components/ui-custom/t.svelte';

	//

	let { data } = $props();
	const { executions } = $derived(data);

	let latestCheckRuns = $state(browser ? LatestCheckRunsStorage.get() : null);
</script>

{#if latestCheckRuns}
	<Alert variant="info" class="mb-8 flex items-center justify-between gap-2">
		<div class="flex items-center gap-3">
			<InfoIcon size={20} />
			<T>
				{m.You_have_count_recent_check_runs({ count: latestCheckRuns.length })}
			</T>
		</div>
		<div class="flex items-center gap-2">
			<Button
				variant="outline"
				onclick={() => {
					LatestCheckRunsStorage.remove();
					latestCheckRuns = null;
				}}
			>
				<X />
				<span>
					{m.Clear_history()}
				</span>
			</Button>
			<Button href="/my/tests/latest">
				<ArrowRightIcon />
				<span>
					{m.Review_latest_check_runs()}
				</span>
			</Button>
		</div>
	</Alert>
{/if}

<TemporalI18nProvider>
	<Table.Root class="rounded-lg bg-white">
		<Table.Header>
			<Table.Row>
				<Table.Head>Status</Table.Head>
				<Table.Head>Workflow ID</Table.Head>
				<Table.Head>Type</Table.Head>
				<Table.Head class="text-right">Start</Table.Head>
				<Table.Head class="text-right">End</Table.Head>
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each executions as workflow (workflow.execution.runId)}
				{@const path = `/my/tests/runs/${workflow.execution.workflowId}/${workflow.execution.runId}`}
				{@const status = toWorkflowStatusReadable(workflow.status)}
				<Table.Row>
					<Table.Cell>
						{#if status !== null}
							<WorkflowStatus {status} />
						{/if}
					</Table.Cell>
					<Table.Cell class="font-medium">
						<A href={path}>{workflow.execution.workflowId}</A>
					</Table.Cell>
					<Table.Cell>{workflow.type.name}</Table.Cell>
					<Table.Cell class="text-right">{toUserTimezone(workflow.startTime)}</Table.Cell>
					<Table.Cell class="text-right">{toUserTimezone(workflow.endTime)}</Table.Cell>
				</Table.Row>
			{:else}
				<Table.Row class="hover:bg-transparent">
					<Table.Cell colspan={5} class="text-center text-gray-300 py-20">
						Test runs will appear here
					</Table.Cell>
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>
</TemporalI18nProvider>
