<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	export type RowSnippet<Workflow extends WorkflowExecutionSummary> = Snippet<
		[
			{
				workflow: Workflow;
				Td: typeof Table.Cell;
			}
		]
	>;
</script>

<script lang="ts" generics="Workflow extends WorkflowExecutionSummary">
	import type { Snippet } from 'svelte';

	import { EllipsisVerticalIcon, TriangleIcon } from '@lucide/svelte';
	import clsx from 'clsx';

	import type { DropdownMenuItem } from '@/components/ui-custom/dropdown-menu.svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import * as Table from '@/components/ui/table';
	import { localizeHref } from '@/i18n';

	import type { WorkflowExecutionSummary } from './queries.types';
	import type { HideColumnsProp } from './workflow-table.types';

	import WorkflowStatusTag from './workflow-status-tag.svelte';
	import WorkflowTableRow from './workflow-table-row.svelte';

	//

	type Props = HideColumnsProp & {
		workflow: Workflow;
		depth?: number;
		row?: RowSnippet<Workflow>;
		disableLink?: boolean;
		actions?: DropdownMenuItem[];
	};

	let {
		workflow,
		depth = 0,
		row,
		hideColumns = [],
		actions,
		disableLink = false
	}: Props = $props();

	const isRoot = $derived(depth === 0);
	const isChild = $derived(!isRoot);

	let isExpanded = $state(true);

	const href = $derived(
		localizeHref(`/my/tests/runs/${workflow.execution.workflowId}/${workflow.execution.runId}`)
	);
</script>

<tr
	class={[
		'hover:bg-transparent',
		{
			'bg-slate-100! text-xs ': isChild,
			'[&>td]:py-0!': isChild,
			'border-b':
				(!isExpanded && isRoot) || !workflow.children || workflow.children.length === 0
		}
	]}
>
	{#if !hideColumns.includes('type')}
		<Table.Cell class="text-muted-foreground">
			<div class="block w-full max-w-[100px] truncate text-ellipsis md:max-w-[180px]">
				{workflow.type.name}
			</div>
		</Table.Cell>
	{/if}

	{#if !hideColumns.includes('workflow')}
		{#if isRoot}
			<Table.Cell class={[isChild && 'py-0!']}>
				<div class="flex flex-wrap items-center gap-4">
					<div class="flex items-center gap-2">
						{#if !disableLink}
							<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
							<a {href} class="text-primary hover:underline">
								{workflow.displayName}
							</a>
						{:else}
							<span>
								{workflow.displayName}
							</span>
						{/if}

						{#if workflow.children && workflow.children.length > 0}
							<Button
								variant="ghost"
								size="icon"
								class="size-6 shrink-0 [&_svg]:size-3"
								onclick={() => (isExpanded = !isExpanded)}
							>
								<TriangleIcon
									class={clsx(
										'fill-primary stroke-none transition-transform duration-200',
										{
											'rotate-180': !isExpanded
										}
									)}
								/>
							</Button>
						{/if}
					</div>
				</div>
			</Table.Cell>
		{:else}
			<Table.Cell
				class={['flex']}
				style="padding-top: 0px!important; padding-bottom: 0px!important"
			>
				<div style={`padding-left: ${(depth - 1) * 16}px`}>
					<div class="border-l border-slate-300 py-2 pl-2">
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
						<a {href} class="text-primary hover:underline">
							{workflow.displayName}
						</a>
					</div>
				</div>
			</Table.Cell>
		{/if}
	{/if}

	{#if !hideColumns.includes('status')}
		<Table.Cell>
			<WorkflowStatusTag
				status={workflow.status}
				failureReason={workflow.failure_reason}
				size={isChild ? 'sm' : 'md'}
			/>
		</Table.Cell>
	{/if}

	{@render row?.({ workflow, Td: Table.Cell })}

	{#if !hideColumns.includes('start_time')}
		<Table.Cell
			class={['text-right', isChild && 'text-[10px] leading-[13px] text-muted-foreground']}
		>
			{workflow.startTime}
		</Table.Cell>
	{/if}

	{#if !hideColumns.includes('end_time')}
		<Table.Cell
			class={[
				'text-right',
				{
					'text-gray-300': !workflow.endTime,
					'text-[10px] leading-[13px] text-muted-foreground': isChild
				}
			]}
		>
			{workflow.endTime ?? 'N/A'}
		</Table.Cell>
	{/if}

	{#if !hideColumns.includes('actions')}
		<Table.Cell class="flex justify-end">
			{#if actions}
				<DropdownMenu items={actions}>
					{#snippet triggerContent()}
						<EllipsisVerticalIcon />
					{/snippet}
				</DropdownMenu>
			{/if}
		</Table.Cell>
	{/if}
</tr>

{#if workflow.children && isExpanded}
	{#each workflow.children as children (children.execution.runId)}
		<WorkflowTableRow workflow={children as Workflow} depth={depth + 1} {row} />
	{/each}
{/if}
