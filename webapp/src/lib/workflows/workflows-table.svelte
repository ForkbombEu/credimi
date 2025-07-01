<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import TemporalI18nProvider from '$routes/my/tests/runs/[workflow_id]/[run_id]/temporal/components/temporal-i18n-provider.svelte';
	import * as Table from '@/components/ui/table';
	import type { WorkflowExecution } from './types';
	import { toWorkflowStatusReadable, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import A from '@/components/ui-custom/a.svelte';
	import { toUserTimezone } from '@/utils/toUserTimezone';
	import { m } from '@/i18n';
	import type { Snippet } from 'svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { getWorkflowMemo, type WorkflowMemo } from './memo';

	//

	type Props = {
		workflows: WorkflowExecution[];
		headerRight?: Snippet<[{ Th: typeof Table.Head }]>;
		rowRight?: Snippet<
			[
				{
					workflow: WorkflowExecution;
					Td: typeof Table.Cell;
					workflowMemo: WorkflowMemo | undefined;
				}
			]
		>;
	};

	let { workflows, headerRight, rowRight }: Props = $props();

	//
</script>

<TemporalI18nProvider>
	<Table.Root class="rounded-lg bg-white">
		<Table.Header>
			<Table.Row>
				<Table.Head>{m.Status()}</Table.Head>
				<Table.Head>{m.Workflow_ID()}</Table.Head>
				{@render headerRight?.({ Th: Table.Head })}
				<Table.Head class="text-right">{m.Start_time()}</Table.Head>
				<Table.Head class="text-right">{m.End_time()}</Table.Head>
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each workflows as workflow (workflow.execution.runId)}
				{@const path = `/my/tests/runs/${workflow.execution.workflowId}/${workflow.execution.runId}`}
				{@const status = toWorkflowStatusReadable(workflow.status)}
				{@const memo = getWorkflowMemo(workflow)}
				{@const start = toUserTimezone(workflow.startTime)}
				{@const end = toUserTimezone(workflow.endTime)}
				<Table.Row>
					<Table.Cell>
						{#if status !== null}
							<WorkflowStatus {status} />
						{/if}
					</Table.Cell>

					<Table.Cell class="font-medium">
						{#if memo}
							<A href={path}>
								<T>{memo.standard} / {memo.author}</T>
								<T>{memo.test}</T>
							</A>
							<T class="mt-1 text-xs text-gray-400">
								{workflow.execution.workflowId}
							</T>
						{:else}
							<A href={path}>
								{workflow.execution.workflowId}
							</A>
						{/if}
					</Table.Cell>

					{@render rowRight?.({ workflow, Td: Table.Cell, workflowMemo: memo })}

					<Table.Cell class="text-right">
						{start}
					</Table.Cell>
					<Table.Cell class={['text-right', { 'text-gray-300': !end }]}>
						{end ?? 'N/A'}
					</Table.Cell>
				</Table.Row>
			{:else}
				<Table.Row class="hover:bg-transparent">
					<Table.Cell colspan={5} class="text-center text-gray-300 py-20">
						{m.Test_runs_will_appear_here()}
					</Table.Cell>
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>
</TemporalI18nProvider>
