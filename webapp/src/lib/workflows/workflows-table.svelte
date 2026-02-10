<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Workflow extends WorkflowExecutionSummary">
	import type { Snippet } from 'svelte';

	import { XIcon } from '@lucide/svelte';
	import { TemporalI18nProvider } from '$lib/temporal';
	import { runWithLoading } from '$lib/utils';

	import type { DropdownMenuItem } from '@/components/ui-custom/dropdown-menu.svelte';

	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import type { WorkflowExecutionSummary } from './queries.types';
	import type { HideColumnsProp } from './workflow-table.types';

	import { cancel } from './utils';
	import WorkflowTableRow, { type RowSnippet } from './workflow-table-row.svelte';

	//

	type Props = HideColumnsProp & {
		workflows: Workflow[];
		header?: Snippet<[{ Th: typeof Table.Head }]>;
		row?: RowSnippet<Workflow>;
		actions?: (workflow: Workflow) => DropdownMenuItem[];
		disableLink?: (workflow: Workflow) => boolean;
	};

	const DEFAULT_ACTIONS = (workflow: WorkflowExecutionSummary): DropdownMenuItem[] => [
		{
			label: m.Cancel(),
			icon: XIcon,
			onclick: () => {
				runWithLoading({
					fn: () => cancel(workflow.execution.workflowId, workflow.execution.runId),
					showSuccessToast: false
				});
			}
		}
	];

	let {
		workflows,
		row,
		header,
		hideColumns = [],
		actions = DEFAULT_ACTIONS,
		disableLink
	}: Props = $props();
</script>

<TemporalI18nProvider>
	<Table.Root class="max-w-full rounded-lg bg-white">
		<Table.Header>
			<Table.Row>
				{#if !hideColumns.includes('type')}
					<Table.Head>{m.Type()}</Table.Head>
				{/if}
				{#if !hideColumns.includes('workflow')}
					<Table.Head>{m.Workflow()}</Table.Head>
				{/if}
				{#if !hideColumns.includes('status')}
					<Table.Head>{m.Status()}</Table.Head>
				{/if}
				{@render header?.({ Th: Table.Head })}
				{#if !hideColumns.includes('start_time')}
					<Table.Head class="text-right">{m.Start_time()}</Table.Head>
				{/if}
				{#if !hideColumns.includes('end_time')}
					<Table.Head class="text-right">{m.End_time()}</Table.Head>
				{/if}
				{#if !hideColumns.includes('actions')}
					<Table.Head class="text-right">{m.Actions()}</Table.Head>
				{/if}
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each workflows as workflow (workflow.execution.runId)}
				<WorkflowTableRow
					{workflow}
					{row}
					{hideColumns}
					actions={actions?.(workflow)}
					disableLink={disableLink?.(workflow)}
				/>
			{:else}
				<Table.Row class="hover:bg-transparent">
					<Table.Cell colspan={6} class="text-center text-gray-300 py-20">
						{m.Test_runs_will_appear_here()}
					</Table.Cell>
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>
</TemporalI18nProvider>
