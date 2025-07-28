<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { TemporalI18nProvider } from '$lib/temporal';
	import * as Table from '@/components/ui/table';
	import { toWorkflowStatusReadable, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import A from '@/components/ui-custom/a.svelte';
	import { toUserTimezone } from '@/utils/toUserTimezone';
	import { m } from '@/i18n';
	import type { Snippet } from 'svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { getWorkflowMemo, type WorkflowMemo } from './memo';
	import { Array } from 'effect';
	import { CornerDownRight } from 'lucide-svelte';
	import type { WorkflowWithChildren } from './utils';

	//

	type Props = {
		workflows: WorkflowWithChildren[];
		headerRight?: Snippet<[{ Th: typeof Table.Head }]>;
		rowRight?: Snippet<
			[
				{
					workflow: WorkflowWithChildren;
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
			{#each workflows as workflow (workflow.runId)}
				{@const path = `/my/tests/runs/${workflow.id}/${workflow.runId}`}
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
								{workflow.id}
							</T>
						{:else}
							<A href={path}>
								{workflow.id}
							</A>
						{/if}

						{#if workflow.children}
							{#each workflow.children as child}
								<div class="pt-2">
									<A
										href={`/my/tests/runs/${child.id}/${child.runId}`}
										class="flex gap-0.5"
									>
										<CornerDownRight size="15" />
										<T class="-translate-y-[1px]">{m.View_logs_workflow()}</T>
									</A>
								</div>
							{/each}
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
