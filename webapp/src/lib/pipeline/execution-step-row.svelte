<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowRightIcon, ChevronDownIcon, FileTextIcon } from '@lucide/svelte';
	import { formatEndClock, splitExecutionTimes } from '$lib/workflows/format-execution-time';

	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';

	import { type ExecutionRef, type ExecutionStep, stepDetailsHref } from './execution-steps';
	import ExecutionSteps from './execution-steps.svelte';
	import TdDash from './td-dash.svelte';
	import WorkflowStatusTag from './workflow-status-tag.svelte';

	type Density = 'compact' | 'comfortable';

	type Props = {
		pipelineId: string;
		run: ExecutionRef;
		step: ExecutionStep;
		depth: number;
		density: Density;
		timezone?: string;
	};

	let { pipelineId, run, step, depth, density, timezone }: Props = $props();

	let expanded = $state(false);

	const row = $derived.by(() => {
		const isCompact = density === 'compact';
		const timeParts = splitExecutionTimes(step.start_time, step.end_time, timezone);
		return {
			isCompact,
			timeParts,
			endClock: timeParts ? formatEndClock(timeParts) : undefined,
			nestPl: depth >= 2 ? `calc(var(--td-px) + ${(depth - 1) * 1.5}rem)` : 'var(--td-px)',
			tdPy: !isCompact && depth > 0 ? '0.25rem' : undefined,
			detailsHref: stepDetailsHref(step),
			iconSize: isCompact ? ('mini' as const) : ('xs' as const)
		};
	});
</script>

<tr
	class={['workflow-row workflow-row-child bg-slate-50', !row.isCompact && 'text-xs']}
	style:--nest-pl={row.nestPl}
	style:--td-py={row.tdPy}
>
	<TdDash
		expand
		when={step.kind === 'block' && Boolean(step.execution) && step.step_count !== 0}
		fallback="empty"
	>
		<Button
			variant="ghost"
			size="sm"
			class={[
				'h-auto gap-0.5 px-2 py-0.5 font-normal',
				row.isCompact ? 'text-xs' : 'text-sm'
			]}
			style={`margin-left: ${depth * 0.75}rem`}
			aria-expanded={expanded}
			aria-label={m.Show_steps()}
			onclick={() => (expanded = !expanded)}
		>
			<ChevronDownIcon
				class={['size-3 transition-transform duration-200', { 'rotate-180': expanded }]}
			/>
			{#if step.step_count !== undefined}
				<span class="text-muted-foreground tabular-nums">{step.step_count}</span>
			{/if}
		</Button>
	</TdDash>

	<TdDash nest>
		<div class="flex items-center gap-1">
			<WorkflowStatusTag status={step.status} failureReason={step.failure_reason} size="sm" />
			{#if step.attempt && step.attempt > 1}
				<span
					class="rounded-sm border px-1 text-[11px] whitespace-nowrap text-muted-foreground tabular-nums"
				>
					{step.max_attempts
						? m.Linked_steps_attempt({
								attempt: step.attempt,
								max_attempts: step.max_attempts
							})
						: m.Step_attempt({ attempt: step.attempt })}
				</span>
			{/if}
		</div>
	</TdDash>

	<TdDash nest colspan={3}>
		<div class="flex min-w-0 items-baseline gap-1.5">
			<span class="shrink-0 font-normal">{step.type}</span>
			<span class="min-w-0 truncate text-muted-foreground">{step.display_name}</span>
			{#if step.has_logs}
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
	<TdDash class="whitespace-nowrap text-muted-foreground" value={step.duration} />

	<TdDash>
		<IconButton
			icon={ArrowRightIcon}
			href={row.detailsHref}
			variant="ghost"
			size={row.iconSize}
			tooltip={m.View()}
			class={['text-primary hover:bg-secondary', row.isCompact && 'rounded-sm']}
			aria-label={m.View()}
		/>
	</TdDash>

	<TdDash fallback="empty" />
</tr>

{#if expanded && step.execution}
	<ExecutionSteps
		{pipelineId}
		{run}
		block={step.execution}
		running={step.status === 'Running'}
		depth={depth + 1}
		{density}
		{timezone}
	/>
{/if}
