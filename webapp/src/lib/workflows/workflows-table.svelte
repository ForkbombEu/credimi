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

	import type { WorkflowExecutionWithChildren } from './queries.types';

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
				{@const status = toWorkflowStatusReadable(workflow.status)}

				<Table.Row>
					<Table.Cell>
						{#if status !== null}
							<WorkflowStatus {status} />
						{/if}
					</Table.Cell>

					<Table.Cell class="font-medium">
						<WorkflowTree {workflow} label={workflow.displayName} root />
					</Table.Cell>

					{@render rowRight?.({ workflow, Td: Table.Cell })}

					<Table.Cell class="text-right">
						{workflow.startTime}
					</Table.Cell>

					<Table.Cell class={['text-right', { 'text-gray-300': !workflow.endTime }]}>
						{workflow.endTime ?? 'N/A'}
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
