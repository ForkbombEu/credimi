<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	export type RowSnippet = Snippet<
		[
			{
				workflow: WorkflowExecutionSummary;
				status: WorkflowStatusType | null;
				Td: typeof Table.Cell;
			}
		]
	>;
</script>

<script lang="ts">
	import type { WorkflowStatusType } from '$lib/temporal';
	import type { Snippet } from 'svelte';

	import { toWorkflowStatusReadable } from '@forkbombeu/temporal-ui';
	import { EllipsisVerticalIcon, ImageIcon, TriangleIcon, VideoIcon } from '@lucide/svelte';
	import clsx from 'clsx';

	import type { IconComponent } from '@/components/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import * as Table from '@/components/ui/table';
	import { localizeHref } from '@/i18n';

	import type { WorkflowExecutionSummary } from './queries.types';

	import WorkflowActions from './workflow-actions.svelte';
	import WorkflowStatus from './workflow-status.svelte';
	import WorkflowTableRow from './workflow-table-row.svelte';

	//

	type Props = {
		workflow: WorkflowExecutionSummary;
		depth?: number;
		hideResults?: boolean;
		row?: RowSnippet;
	};

	let { workflow, depth = 0, row, hideResults = false }: Props = $props();

	const hasQueue = $derived(!!workflow.queue);
	const status = $derived(
		hasQueue ? null : (toWorkflowStatusReadable(workflow.status) as WorkflowStatusType)
	);

	const isRoot = $derived(depth === 0);
	const isChild = $derived(!isRoot);

	let isExpanded = $state(true);

	const href = $derived(
		hasQueue
			? null
			: localizeHref(`/my/tests/runs/${workflow.execution.workflowId}/${workflow.execution.runId}`)
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
	<Table.Cell class="text-muted-foreground">
		<div class="block w-full max-w-[100px] truncate text-ellipsis md:max-w-[180px]">
			{workflow.type.name}
		</div>
	</Table.Cell>

	{#if isRoot}
		<Table.Cell class={[isChild && 'py-0!']}>
			<div class="flex flex-wrap items-center gap-4">
				<div class="flex items-center gap-2">
					{#if href}
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
						<a {href} class="text-primary hover:underline">
							{workflow.displayName}
						</a>
					{:else}
						<span>{workflow.displayName}</span>
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
					{#if href}
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
						<a {href} class="text-primary hover:underline">
							{workflow.displayName}
						</a>
					{:else}
						<span>{workflow.displayName}</span>
					{/if}
				</div>
			</div>
		</Table.Cell>
	{/if}

	<Table.Cell>
		<WorkflowStatus
			{status}
			failureReason={workflow.failure_reason}
			queue={workflow.queue}
			size={isChild ? 'sm' : 'md'}
		/>
	</Table.Cell>

	{@render row?.({ workflow, Td: Table.Cell, status })}

	{#if !hideResults}
		<Table.Cell>
			{#if workflow.results && workflow.results.length > 0}
				<div class="flex items-center gap-2">
					{#each workflow.results as result (result.video)}
						<div class="flex items-center gap-1">
							{@render mediaPreview({
								image: result.screenshot,
								href: result.video,
								icon: VideoIcon
							})}
							{@render mediaPreview({
								image: result.screenshot,
								href: result.screenshot,
								icon: ImageIcon
							})}
						</div>
					{/each}
				</div>
			{:else}
				<span class="text-muted-foreground opacity-50">N/A</span>
			{/if}
		</Table.Cell>
	{/if}

	<Table.Cell
		class={['text-right', isChild && 'text-[10px] leading-[13px] text-muted-foreground']}
	>
		{workflow.startTime}
	</Table.Cell>

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

	<Table.Cell class="flex justify-end">
		<WorkflowActions
			mode="dropdown"
			workflow={{
				workflowId: workflow.execution.workflowId,
				runId: workflow.execution.runId,
				status: status,
				name: workflow.displayName,
				queue: workflow.queue
			}}
			dropdownTriggerVariants={{ size: 'icon', variant: 'ghost' }}
		>
			{#snippet dropdownTriggerContent()}
				<EllipsisVerticalIcon />
			{/snippet}
		</WorkflowActions>
	</Table.Cell>
</tr>

{#if workflow.children && isExpanded}
	{#each workflow.children as children (children.queue?.ticket_id ?? children.execution.runId)}
		<WorkflowTableRow workflow={children} depth={depth + 1} {row} />
	{/each}
{/if}

{#snippet mediaPreview(props: { image: string; href: string; icon: IconComponent })}
	{@const { image, href, icon } = props}
	<!-- eslint-disable svelte/no-navigation-without-resolve -->
	<a
		{href}
		target="_blank"
		class="relative size-10 shrink-0 overflow-hidden rounded-md border border-slate-300 hover:cursor-pointer hover:ring-2"
	>
		<img src={image} alt="Media" class="size-10 shrink-0" />
		<div class="absolute inset-0 flex items-center justify-center bg-black/30">
			<Icon src={icon} class="size-4  text-white" />
		</div>
	</a>
{/snippet}
