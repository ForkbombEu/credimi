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

	let { data } = $props();
	const { executions } = $derived(data);
</script>

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
