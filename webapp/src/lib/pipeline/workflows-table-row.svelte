<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SvelteSet } from 'svelte/reactivity';

	import {
		ArrowRightIcon,
		ChevronDownIcon,
		EllipsisVerticalIcon,
		FileTextIcon
	} from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { formatEndClock, splitExecutionTimes } from '$lib/workflows/format-execution-time';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';

	import type { ExecutionSummary } from './workflows';

	import { makeDropdownActions } from './actions';
	import { fromApiSummary } from './execution-artifacts';
	import ExecutionDevices from './execution-devices.svelte';
	import ExecutionProgress from './execution-progress.svelte';
	import ExecutionArtifactsPreview from './results/execution-artifacts-preview.svelte';
	import TdDash from './td-dash.svelte';
	import WorkflowStatusTag from './workflow-status-tag.svelte';
	import WorkflowsTableRow from './workflows-table-row.svelte';

	type Density = 'compact' | 'comfortable';

	type Props = {
		workflow: ExecutionSummary;
		depth?: number;
		density: Density;
		timezone?: string;
		expandedRunIds: SvelteSet<string>;
		onToggle: (runId: string) => void;
		onCancel?: () => void;
		/** Comfortable root rows after the first get a hairline separator. */
		showTopBorder?: boolean;
	};

	let {
		workflow,
		depth = 0,
		density,
		timezone,
		expandedRunIds,
		onToggle,
		onCancel,
		showTopBorder = false
	}: Props = $props();

	const row = $derived.by(() => {
		const isRoot = depth === 0;
		const isCompact = density === 'compact';
		const children = (workflow.children ?? []) as ExecutionSummary[];
		const childCount = children.length;
		const timeParts =
			isRoot && workflow.queue
				? workflow.enqueuedAt
					? splitExecutionTimes(workflow.enqueuedAt, undefined, timezone)
					: undefined
				: splitExecutionTimes(workflow.startTime, workflow.endTime, timezone);

		return {
			isRoot,
			isCompact,
			children,
			childCount,
			isExpanded: expandedRunIds.has(workflow.execution.runId) && childCount > 0,
			artifacts: isRoot ? fromApiSummary(workflow) : undefined,
			timeParts,
			endClock: timeParts ? formatEndClock(timeParts) : undefined,
			nestPl: depth >= 2 ? `calc(var(--td-px) + ${(depth - 1) * 1.5}rem)` : 'var(--td-px)',
			tdPy: !isCompact && depth > 0 ? '0.25rem' : undefined,
			detailsHref: resolve('/my/tests/runs/[workflow_id]/[run_id]', {
				workflow_id: workflow.execution.workflowId,
				run_id: workflow.execution.runId
			}),
			artifactIconSize: isCompact ? ('mini' as const) : ('xs' as const),
			artifactIconClass: isCompact
				? 'text-primary hover:bg-secondary'
				: 'rounded-sm text-primary hover:bg-secondary'
		};
	});
</script>

<tr
	class={[
		'workflow-row',
		row.isRoot ? 'workflow-row-root' : 'workflow-row-child bg-slate-50',
		!row.isCompact && depth > 0 && 'text-xs',
		showTopBorder && '[&>td]:border-t [&>td]:border-border/60'
	]}
	style:--nest-pl={row.nestPl}
	style:--td-py={row.tdPy}
>
	<TdDash expand when={row.childCount > 0} fallback={row.isRoot ? 'dash' : 'empty'}>
		<Button
			variant="ghost"
			size="sm"
			class={[
				'h-auto gap-0.5 px-2 py-0.5 font-normal',
				row.isCompact ? 'text-xs' : 'text-sm'
			]}
			aria-expanded={row.isExpanded}
			aria-label={m.count_children({ count: row.childCount })}
			onclick={() => onToggle(workflow.execution.runId)}
		>
			<ChevronDownIcon
				class={[
					'size-3 transition-transform duration-200',
					{ 'rotate-180': row.isExpanded }
				]}
			/>
			<span class="text-muted-foreground tabular-nums">{row.childCount}</span>
		</Button>
	</TdDash>

	<TdDash nest>
		<WorkflowStatusTag
			status={workflow.status}
			queueData={row.isRoot ? workflow.queue : undefined}
			failureReason={workflow.failure_reason}
			size={row.isRoot && !row.isCompact ? 'md' : 'sm'}
		/>
	</TdDash>

	{#if row.isRoot}
		<TdDash>
			<ExecutionDevices execution={workflow} compact={row.isCompact} />
		</TdDash>
		<TdDash when={Boolean(row.artifacts)}>
			{#if row.artifacts}
				<ExecutionArtifactsPreview
					artifacts={row.artifacts}
					presentation={row.isCompact ? 'icons-sm' : 'icons'}
				/>
			{/if}
		</TdDash>
		<TdDash class="whitespace-nowrap text-muted-foreground" value={row.timeParts?.date} />
		<TdDash class="whitespace-nowrap text-muted-foreground" value={row.timeParts?.start} />
		{#if workflow.status === 'Running'}
			<TdDash colspan={2}>
				<ExecutionProgress progress={workflow.progress} />
			</TdDash>
		{:else}
			<TdDash class="whitespace-nowrap text-muted-foreground" value={row.endClock} />
			<TdDash class="whitespace-nowrap text-muted-foreground" value={workflow.duration} />
		{/if}
	{:else}
		<TdDash nest colspan={3}>
			<div class="flex min-w-0 items-baseline gap-1.5">
				<span class="shrink-0 font-normal">{workflow.type.name}</span>
				<span class="min-w-0 truncate text-muted-foreground">{workflow.displayName}</span>
				{#if workflow.has_logs}
					<Tooltip>
						{#snippet child({ props })}
							<span
								{...props}
								class="inline-flex shrink-0 -translate-x-px translate-y-px"
							>
								<FileTextIcon size={12} class="text-muted-foreground" />
							</span>
						{/snippet}
						{#snippet content()}
							<p>{m.pipeline_artifact_log_tooltip()}</p>
						{/snippet}
					</Tooltip>
				{/if}
			</div>
		</TdDash>
		<TdDash class="whitespace-nowrap text-muted-foreground" value={row.timeParts?.start} />
		<TdDash class="whitespace-nowrap text-muted-foreground" value={row.endClock} />
		<TdDash class="whitespace-nowrap text-muted-foreground" value={workflow.duration} />
	{/if}

	<TdDash when={!(row.isRoot && workflow.queue)}>
		<IconButton
			icon={ArrowRightIcon}
			href={row.detailsHref}
			variant="ghost"
			size={row.artifactIconSize}
			tooltip={m.View()}
			class={row.artifactIconClass}
			aria-label={m.View()}
		/>
	</TdDash>

	<TdDash class="text-right">
		<DropdownMenu
			items={makeDropdownActions(workflow, { onSettled: onCancel })}
			triggerVariants={{ variant: 'ghost', size: 'icon-sm' }}
		>
			{#snippet trigger({ props })}
				<IconButton {...props} icon={EllipsisVerticalIcon} variant="ghost" size="xs" />
			{/snippet}
		</DropdownMenu>
	</TdDash>
</tr>

{#if row.isExpanded}
	{#each row.children as child (child.execution.runId)}
		<WorkflowsTableRow
			workflow={child}
			depth={depth + 1}
			{density}
			{timezone}
			{expandedRunIds}
			{onToggle}
			{onCancel}
		/>
	{/each}
{/if}
