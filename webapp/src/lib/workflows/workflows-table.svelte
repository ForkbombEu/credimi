<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { toWorkflowStatusReadable, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { TemporalI18nProvider } from '$lib/temporal';

	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';
	import { toUserTimezone } from '@/utils/toUserTimezone';

	import type { WorkflowExecutionWithChildren } from './queries.types';

	import { getWorkflowMemo, type WorkflowMemo } from './memo';
	import WorkflowTree from './workflow-tree.svelte';

	//

	type Props = {
		workflows: WorkflowExecutionWithChildren[];
		headerRight?: Snippet<[{ Th: typeof Table.Head }]>;
		rowRight?: Snippet<
			[
				{
					workflow: WorkflowExecutionWithChildren;
					Td: typeof Table.Cell;
					workflowMemo: WorkflowMemo | undefined;
				}
			]
		>;
	};

	let { workflows, headerRight, rowRight }: Props = $props();
</script>

<TemporalI18nProvider>
	<Table.Root class="rounded-lg bg-white">
		<Table.Header>
			<Table.Row>
				<Table.Head>{m.Status()}</Table.Head>
				<Table.Head>{m.Workflow()}</Table.Head>
				{@render headerRight?.({ Th: Table.Head })}
				<Table.Head class="text-right">{m.Start_time()}</Table.Head>
				<Table.Head class="text-right">{m.End_time()}</Table.Head>
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each workflows as workflow (workflow.execution.runId)}
				{@const { execution } = workflow}
				{@const status = toWorkflowStatusReadable(workflow.status)}
				{@const memo = getWorkflowMemo(workflow)}
				{@const start = toUserTimezone(workflow.startTime)}
				{@const end = toUserTimezone(workflow.endTime)}

				<Table.Row>
					<Table.Cell class="align-top">
						{#if status !== null}
							<WorkflowStatus {status} />
						{/if}
					</Table.Cell>

					<Table.Cell class="font-medium">
						{@const label = memo
							? `${memo.standard} / ${memo.author}`
							: execution.workflowId}
						<WorkflowTree {workflow} {label} root />
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
