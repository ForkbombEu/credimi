<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SvelteSet } from 'svelte/reactivity';

	import { resolve } from '$app/paths';
	import { formatEndClock, splitExecutionTimes } from '$lib/workflows/format-execution-time';
	import {
		ArrowRightIcon,
		ChevronDownIcon,
		EllipsisVerticalIcon,
		FileTextIcon
	} from '@lucide/svelte';

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
	import TdDash, { cellPadClass } from './td-dash.svelte';
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

	const isRoot = $derived(depth === 0);
	const isChild = $derived(depth > 0);
	const isCompact = $derived(density === 'compact');
	const children = $derived((workflow.children ?? []) as ExecutionSummary[]);
	const childCount = $derived(children.length);
	const isExpanded = $derived(expandedRunIds.has(workflow.execution.runId) && childCount > 0);

	const artifacts = $derived(isRoot ? fromApiSummary(workflow) : undefined);

	const timeParts = $derived.by(() => {
		if (isRoot && workflow.queue) {
			if (!workflow.enqueuedAt) return undefined;
			return splitExecutionTimes(workflow.enqueuedAt, undefined, timezone);
		}
		return splitExecutionTimes(workflow.startTime, workflow.endTime, timezone);
	});
	const endClock = $derived(timeParts ? formatEndClock(timeParts) : undefined);

	/**
	 * Nest status + child name from depth 2 up (depth 0/1 flush).
	 * depth 2 → 1.5rem, depth 3 → 3rem, …
	 */
	const nestIndent = $derived(depth >= 2 ? `${(depth - 1) * 1.5}rem` : undefined);
	const pad = $derived(cellPadClass(density, { child: isChild }));
	const detailsHref = $derived(
		resolve('/my/tests/runs/[workflow_id]/[run_id]', {
			workflow_id: workflow.execution.workflowId,
			run_id: workflow.execution.runId
		})
	);
	const artifactIconSize = $derived(isCompact ? 'mini' : 'xs');
	const artifactIconClass = $derived(
		isCompact ? 'text-primary hover:bg-secondary' : 'rounded-sm text-primary hover:bg-secondary'
	);
</script>

<tr
	class={[
		'workflow-row',
		isRoot ? 'workflow-row-root' : 'workflow-row-child bg-slate-50',
		showTopBorder && '[&>td]:border-t [&>td]:border-border/60'
	]}
>
	<TdDash
		class="col-expand"
		{density}
		child={isChild}
		expand
		when={childCount > 0}
		fallback={isRoot ? 'dash' : 'empty'}
	>
		<Button
			variant="ghost"
			size="sm"
			class={['h-auto gap-0.5 px-2 py-0.5 font-normal', isCompact ? 'text-xs' : 'text-sm']}
			aria-expanded={isExpanded}
			aria-label={m.count_children({ count: childCount })}
			onclick={() => onToggle(workflow.execution.runId)}
		>
			<ChevronDownIcon
				class={['size-3 transition-transform duration-200', { 'rotate-180': isExpanded }]}
			/>
			<span class="text-muted-foreground tabular-nums">{childCount}</span>
		</Button>
	</TdDash>

	<td class={pad} style:padding-left={nestIndent}>
		<WorkflowStatusTag
			status={workflow.status}
			queueData={isRoot ? workflow.queue : undefined}
			failureReason={workflow.failure_reason}
			size={isRoot && !isCompact ? 'md' : 'sm'}
		/>
	</td>

	{#if isRoot}
		<td class={pad}>
			<ExecutionDevices execution={workflow} compact={isCompact} />
		</td>
		<TdDash {density} when={Boolean(artifacts)}>
			{#if artifacts}
				<ExecutionArtifactsPreview
					{artifacts}
					presentation={isCompact ? 'icons-sm' : 'icons'}
				/>
			{/if}
		</TdDash>
		<TdDash {density} class="whitespace-nowrap text-muted-foreground" value={timeParts?.date} />
		<TdDash
			{density}
			class="whitespace-nowrap text-muted-foreground"
			value={timeParts?.start}
		/>
		{#if workflow.status === 'Running'}
			<td class={pad} colspan="2">
				<ExecutionProgress progress={workflow.progress} />
			</td>
		{:else}
			<TdDash {density} class="whitespace-nowrap text-muted-foreground" value={endClock} />
			<TdDash
				{density}
				class="whitespace-nowrap text-muted-foreground"
				value={workflow.duration}
			/>
		{/if}
	{:else}
		<td class={pad} colspan="3" style:padding-left={nestIndent}>
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
		</td>
		<TdDash
			{density}
			child
			class="whitespace-nowrap text-muted-foreground"
			value={timeParts?.start}
		/>
		<TdDash {density} child class="whitespace-nowrap text-muted-foreground" value={endClock} />
		<TdDash
			{density}
			child
			class="whitespace-nowrap text-muted-foreground"
			value={workflow.duration}
		/>
	{/if}

	<TdDash {density} child={isChild} when={!(isRoot && workflow.queue)}>
		<IconButton
			icon={ArrowRightIcon}
			href={detailsHref}
			variant="ghost"
			size={artifactIconSize}
			tooltip={m.View()}
			class={artifactIconClass}
			aria-label={m.View()}
		/>
	</TdDash>

	<td class={['text-right', pad]}>
		<DropdownMenu
			items={makeDropdownActions(workflow, { onSettled: onCancel })}
			triggerVariants={{ variant: 'ghost', size: 'icon-sm' }}
		>
			{#snippet trigger({ props })}
				<IconButton {...props} icon={EllipsisVerticalIcon} variant="ghost" size="xs" />
			{/snippet}
		</DropdownMenu>
	</td>
</tr>

{#if isExpanded}
	{#each children as child (child.execution.runId)}
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
