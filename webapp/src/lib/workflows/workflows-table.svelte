<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { TemporalI18nProvider } from '$lib/temporal';

	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import type { WorkflowExecutionSummary } from './queries.types';

	import WorkflowTableRow from './workflow-table-row.svelte';

	//

	type Props = {
		workflows: WorkflowExecutionSummary[];
		hideResults?: boolean;
		row?: Snippet<[{ workflow: WorkflowExecutionSummary }]>;
		header?: Snippet<[{ Th: typeof Table.Head }]>;
	};

	let { workflows, row, header, hideResults = false }: Props = $props();
</script>

<TemporalI18nProvider>
	<Table.Root class="max-w-full rounded-lg bg-white">
		<Table.Header>
			<Table.Row>
				<Table.Head>{m.Type()}</Table.Head>
				<Table.Head>{m.Workflow()}</Table.Head>
				<Table.Head>{m.Status()}</Table.Head>
				{#if !hideResults}
					<Table.Head>{m.Results()}</Table.Head>
				{/if}
				{@render header?.({ Th: Table.Head })}
				<Table.Head class="text-right">{m.Start_time()}</Table.Head>
				<Table.Head class="text-right">{m.End_time()}</Table.Head>
				<Table.Head class="text-right">{m.Actions()}</Table.Head>
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each workflows as workflow (workflow.execution.runId)}
				<WorkflowTableRow {workflow} {row} {hideResults} />
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
