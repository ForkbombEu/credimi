<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SvelteSet } from 'svelte/reactivity';

	import { ArrowRightIcon, ChevronDownIcon, EllipsisVerticalIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { formatEndClock, splitExecutionTimes } from '$lib/workflows/format-execution-time';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import type { ExecutionSummary } from './workflows';

	import { makeDropdownActions } from './actions';
	import { fromApiSummary } from './execution-artifacts';
	import ExecutionDevices from './execution-devices.svelte';
	import ExecutionProgress from './execution-progress.svelte';
	import ExecutionSteps from './execution-steps.svelte';
	import ExecutionArtifactsPreview from './results/execution-artifacts-preview.svelte';
	import TdDash from './td-dash.svelte';
	import WorkflowStatusTag from './workflow-status-tag.svelte';

	type Density = 'compact' | 'comfortable';

	type Props = {
		pipelineId: string;
		workflow: ExecutionSummary;
		density: Density;
		timezone?: string;
		expandedRunIds: SvelteSet<string>;
		onToggle: (runId: string) => void;
		onCancel?: () => void;
		/** Comfortable root rows after the first get a hairline separator. */
		showTopBorder?: boolean;
	};

	let {
		pipelineId,
		workflow,
		density,
		timezone,
		expandedRunIds,
		onToggle,
		onCancel,
		showTopBorder = false
	}: Props = $props();

	const row = $derived.by(() => {
		const isCompact = density === 'compact';
		const timeParts = workflow.queue
			? workflow.enqueuedAt
				? splitExecutionTimes(workflow.enqueuedAt, undefined, timezone)
				: undefined
			: splitExecutionTimes(workflow.startTime, workflow.endTime, timezone);

		// An unknown count (history not readable) still lets the user try to expand.
		const hasSteps = !workflow.queue && workflow.step_count !== 0;

		return {
			isCompact,
			hasSteps,
			isExpanded: expandedRunIds.has(workflow.execution.runId) && hasSteps,
			artifacts: fromApiSummary(workflow),
			timeParts,
			endClock: timeParts ? formatEndClock(timeParts) : undefined,
			detailsHref: resolve('/my/tests/runs/[workflow_id]/[run_id]', {
				workflow_id: workflow.execution.workflowId,
				run_id: workflow.execution.runId
			}),
			artifactIconSize: isCompact ? ('mini' as const) : ('xs' as const)
		};
	});
</script>

<tr
	class={[
		'workflow-row workflow-row-root',
		showTopBorder && '[&>td]:border-t [&>td]:border-border/60'
	]}
	style:--nest-pl="var(--td-px)"
>
	<TdDash expand when={row.hasSteps}>
		<Button
			variant="ghost"
			size="sm"
			class={[
				'h-auto gap-0.5 px-2 py-0.5 font-normal',
				row.isCompact ? 'text-xs' : 'text-sm'
			]}
			aria-expanded={row.isExpanded}
			aria-label={m.Show_steps()}
			onclick={() => onToggle(workflow.execution.runId)}
		>
			<ChevronDownIcon
				class={[
					'size-3 transition-transform duration-200',
					{ 'rotate-180': row.isExpanded }
				]}
			/>
			{#if workflow.step_count !== undefined}
				<span class="text-muted-foreground tabular-nums">{workflow.step_count}</span>
			{/if}
		</Button>
	</TdDash>

	<TdDash nest>
		<WorkflowStatusTag
			status={workflow.status}
			queueData={workflow.queue}
			failureReason={workflow.failure_reason}
			size={row.isCompact ? 'sm' : 'md'}
		/>
	</TdDash>

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

	<TdDash when={!workflow.queue}>
		<IconButton
			icon={ArrowRightIcon}
			href={row.detailsHref}
			variant="ghost"
			size={row.artifactIconSize}
			tooltip={m.View()}
			class={['text-primary hover:bg-secondary', row.isCompact && 'rounded-sm']}
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
	<ExecutionSteps
		{pipelineId}
		run={workflow.execution}
		running={workflow.status === 'Running'}
		depth={1}
		{density}
		{timezone}
	/>
{/if}
