<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import TemporalI18nProvider from '$routes/my/tests/runs/[workflow_id]/[run_id]/components/temporal-i18n-provider.svelte';
	import * as Table from '@/components/ui/table';
	import type { WorkflowExecution } from './types';
	import { toWorkflowStatusReadable, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import A from '@/components/ui-custom/a.svelte';
	import { toUserTimezone } from '@/utils/toUserTimezone';
	import { m } from '@/i18n';

	type Props = {
		workflows: WorkflowExecution[];
	};

	let { workflows }: Props = $props();
</script>

<TemporalI18nProvider>
	<Table.Root class="rounded-lg bg-white">
		<Table.Header>
			<Table.Row>
				<Table.Head>{m.Status()}</Table.Head>
				<Table.Head>{m.Workflow_ID()}</Table.Head>
				<Table.Head>{m.Type()}</Table.Head>
				<Table.Head class="text-right">{m.Start_time()}</Table.Head>
				<Table.Head class="text-right">{m.End_time()}</Table.Head>
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each workflows as workflow (workflow.execution.runId)}
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
						{m.Test_runs_will_appear_here()}
					</Table.Cell>
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>
</TemporalI18nProvider>
