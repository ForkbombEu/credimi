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
	import type { Snippet } from 'svelte';
	import { z } from 'zod';
	import T from '@/components/ui-custom/t.svelte';

	//

	type Props = {
		workflows: WorkflowExecution[];
		headerRight?: Snippet<[{ Th: typeof Table.Head }]>;
		rowRight?: Snippet<[{ workflow: WorkflowExecution; Td: typeof Table.Cell }]>;
	};

	let { workflows, headerRight, rowRight }: Props = $props();

	//

	const memoFieldSchema = z.object({
		data: z.string(),
		metadata: z.object({
			encoding: z.string()
		})
	});

	type MemoField = z.infer<typeof memoFieldSchema>;

	function getWorkflowMemo(workflow: WorkflowExecution) {
		try {
			const fields = z.record(memoFieldSchema).parse(workflow.memo['fields']);
			if (!fields) return undefined;
			const author = memoFieldToText(fields['author']);
			const standard = memoFieldToText(fields['standard']);
			const test = memoFieldToText(fields['test'])?.split('/').at(-1)?.split('.').at(0);
			if (!author || !standard || !test) return undefined;
			return {
				author,
				standard,
				test
			};
		} catch (error) {
			console.warn(`Failed to parse memo: ${error}`);
			return undefined;
		}
	}

	function memoFieldToText(field: MemoField | undefined) {
		if (!field) return undefined;
		try {
			const { data } = field;
			return atob(data).replaceAll('"', '').trim();
		} catch (error) {
			throw new Error(`Failed to decode memo field: ${error}`);
		}
	}
</script>

<TemporalI18nProvider>
	<Table.Root class="rounded-lg bg-white">
		<Table.Header>
			<Table.Row>
				<Table.Head>{m.Status()}</Table.Head>
				<Table.Head>{m.Workflow_ID()}</Table.Head>
				<Table.Head class="text-right">{m.Start_time()}</Table.Head>
				<Table.Head class="text-right">{m.End_time()}</Table.Head>
				{@render headerRight?.({ Th: Table.Head })}
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each workflows as workflow (workflow.execution.runId)}
				{@const path = `/my/tests/runs/${workflow.execution.workflowId}/${workflow.execution.runId}`}
				{@const status = toWorkflowStatusReadable(workflow.status)}
				{@const memo = getWorkflowMemo(workflow)}
				<Table.Row>
					<Table.Cell>
						{#if status !== null}
							<WorkflowStatus {status} />
						{/if}
					</Table.Cell>

					<Table.Cell class="font-medium">
						<A href={path}>
							{#if memo}
								<T>{memo.standard}</T>
								<T>{memo.author}</T>
								<T>{memo.test}</T>
							{:else}
								{workflow.execution.workflowId}
							{/if}
						</A>
					</Table.Cell>

					<Table.Cell class="text-right">{toUserTimezone(workflow.startTime)}</Table.Cell>
					<Table.Cell class="text-right">{toUserTimezone(workflow.endTime)}</Table.Cell>
					{@render rowRight?.({ workflow, Td: Table.Cell })}
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
