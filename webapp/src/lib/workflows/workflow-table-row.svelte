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
				depth: number;
			}
		]
	>;
</script>

<script lang="ts" generics="Workflow extends WorkflowExecutionSummary">
	import { ChevronUp, EllipsisVerticalIcon, FileTextIcon } from '@lucide/svelte';
	import { formatEndClock, splitExecutionTimes } from '$lib/workflows/format-execution-time';
	import clsx from 'clsx';
	import { String } from 'effect';
	import { untrack, type Snippet } from 'svelte';
	import { fromStore } from 'svelte/store';

	import type { DropdownMenuItem } from '@/components/ui-custom/dropdown-menu.svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import * as Table from '@/components/ui/table';
	import { localizeHref, m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import type { WorkflowExecutionSummary } from './queries.types';
	import type { HideColumnsProp } from './workflow-table.types';

	import WorkflowStatusTag from './workflow-status-tag.svelte';
	import WorkflowTableRow from './workflow-table-row.svelte';

	//

	type Props = HideColumnsProp & {
		workflow: Workflow;
		depth?: number;
		row?: RowSnippet<Workflow>;
		disableLink?: (workflow: Workflow) => boolean;
		actions?: (workflow: Workflow) => DropdownMenuItem[];
		rowStart?: RowSnippet<Workflow>;
		defaultExpanded?: boolean;
	};

	let {
		workflow,
		depth = 0,
		row,
		hideColumns = [],
		actions,
		disableLink,
		rowStart,
		defaultExpanded = false
	}: Props = $props();

	const user = fromStore(currentUser);
	const timezone = $derived(user.current?.Timezone);

	const isRoot = $derived(depth === 0);
	const isChild = $derived(!isRoot);
	const hasChildren = $derived((workflow.children?.length ?? 0) > 0);

	let isExpanded = $state(untrack(() => defaultExpanded));

	const href = $derived(
		localizeHref(`/my/tests/runs/${workflow.execution.workflowId}/${workflow.execution.runId}`)
	);

	const timeParts = $derived.by(() => {
		if (workflow.queue) {
			if (!workflow.enqueuedAt) return undefined;
			return splitExecutionTimes(workflow.enqueuedAt, undefined, timezone);
		}
		return splitExecutionTimes(workflow.startTime, workflow.endTime, timezone);
	});
	const endClock = $derived(timeParts ? formatEndClock(timeParts) : undefined);
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
	{@render rowStart?.({ workflow, Td: Table.Cell, depth })}

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
					<div class="flex items-center gap-0.5">
						{#if hasChildren}
							<Button
								variant="ghost"
								size="icon"
								class="size-6 shrink-0 [&_svg]:size-3"
								onclick={() => (isExpanded = !isExpanded)}
							>
								<ChevronUp
									class={clsx('transition-transform duration-200', {
										'rotate-180': !isExpanded
									})}
								/>
							</Button>
						{/if}

						{#if !disableLink?.(workflow)}
							<a {href} class="text-primary hover:underline">
								{workflow.displayName}
							</a>
						{:else}
							<span>
								{workflow.displayName}
							</span>
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
					<div class="py-2 pl-2">
						<div class="flex min-w-0 items-center gap-0.5">
							{#if hasChildren}
								<Button
									variant="ghost"
									size="icon"
									class="size-6 shrink-0 [&_svg]:size-3"
									onclick={() => (isExpanded = !isExpanded)}
								>
									<ChevronUp
										class={clsx('transition-transform duration-200', {
											'rotate-180': !isExpanded
										})}
									/>
								</Button>
							{/if}
							<div class="flex min-w-0 items-baseline gap-1.5">
								<a {href} class="text-primary hover:underline">
									{workflow.displayName}
								</a>
								{#if workflow.has_logs}
									<Tooltip>
										{#snippet child({ props })}
											<span
												{...props}
												class="inline-flex shrink-0 -translate-x-px translate-y-px"
											>
												<FileTextIcon
													size={12}
													class="text-muted-foreground"
												/>
											</span>
										{/snippet}
										{#snippet content()}
											<p>{m.pipeline_artifact_log_tooltip()}</p>
										{/snippet}
									</Tooltip>
								{/if}
							</div>
						</div>
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

	{@render row?.({ workflow, Td: Table.Cell, depth })}

	{#if !hideColumns.includes('date')}
		<Table.Cell
			class={['text-right', isChild && 'text-[10px] leading-[13px] text-muted-foreground']}
		>
			{@render na(timeParts?.date)}
		</Table.Cell>
	{/if}

	{#if !hideColumns.includes('start')}
		<Table.Cell
			class={['text-right', isChild && 'text-[10px] leading-[13px] text-muted-foreground']}
		>
			{@render na(timeParts?.start)}
		</Table.Cell>
	{/if}

	{#if !hideColumns.includes('end')}
		<Table.Cell
			class={['text-right', isChild && 'text-[10px] leading-[13px] text-muted-foreground']}
		>
			{@render na(endClock)}
		</Table.Cell>
	{/if}

	{#if !hideColumns.includes('duration')}
		<Table.Cell
			class={['text-right', isChild && 'text-[10px] leading-[13px] text-muted-foreground']}
		>
			{@render na(workflow.duration)}
		</Table.Cell>
	{/if}

	{#if !hideColumns.includes('actions')}
		<Table.Cell>
			<div class="flex justify-end">
				{#if actions}
					<DropdownMenu
						items={actions(workflow)}
						triggerVariants={{ variant: 'ghost', size: 'icon-sm' }}
					>
						{#snippet triggerContent()}
							<EllipsisVerticalIcon />
						{/snippet}
					</DropdownMenu>
				{/if}
			</div>
		</Table.Cell>
	{/if}
</tr>

{#if workflow.children && isExpanded}
	{#each workflow.children as child, index (child.execution.runId)}
		<WorkflowTableRow
			workflow={child as Workflow}
			depth={depth + 1}
			{row}
			{hideColumns}
			{actions}
			{disableLink}
			{rowStart}
			defaultExpanded={child.status === 'Running' ||
				index === (workflow.children?.length ?? 0) - 1}
		/>
	{/each}
{/if}

{#snippet na(value: string | undefined)}
	{#if value && String.isNonEmpty(value)}
		{value}
	{:else}
		<span class="text-muted-foreground opacity-50">N/A</span>
	{/if}
{/snippet}
