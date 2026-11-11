<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { RefreshCw } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';

	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import { m } from '@/i18n';

	import ExecutionStepRow from './execution-step-row.svelte';
	import { type ExecutionRef, listExecutionSteps, visibleSteps } from './execution-steps';

	type Density = 'compact' | 'comfortable';

	type Props = {
		pipelineId: string;
		run: ExecutionRef;
		/** A linked steps block run of `run`: lists the steps inside it. */
		block?: ExecutionRef;
		running: boolean;
		depth: number;
		density: Density;
		timezone?: string;
	};

	let { pipelineId, run, block, running, depth, density, timezone }: Props = $props();

	const steps = createQuery(() => ({
		queryKey: [
			'pipeline-execution-steps',
			pipelineId,
			run.workflowId,
			run.runId,
			block?.workflowId ?? '',
			block?.runId ?? ''
		] as const,
		queryFn: () => listExecutionSteps(pipelineId, run, block),
		refetchInterval: running ? 5000 : false,
		staleTime: running ? 0 : Infinity
	}));

	let showAll = $state(false);

	const shown = $derived(steps.data ? visibleSteps(steps.data, showAll) : []);
	const nestPl = $derived(
		depth >= 2 ? `calc(var(--td-px) + ${(depth - 1) * 1.5}rem)` : 'var(--td-px)'
	);
	const tdPy = $derived(density === 'comfortable' ? '0.25rem' : undefined);
</script>

{#if steps.isSuccess && steps.data.length > 0}
	{#each shown as step (step.history_event.event_id)}
		<ExecutionStepRow {pipelineId} {run} {step} {depth} {density} {timezone} />
	{/each}
	{#if shown.length < steps.data.length}
		<tr class="workflow-row-child bg-slate-50" style:--nest-pl={nestPl} style:--td-py={tdPy}>
			<td colspan={10} class="py-(--td-py) pr-(--td-px) pl-(--nest-pl)">
				<Button
					variant="link"
					size="sm"
					class="h-auto px-0 py-0"
					onclick={() => (showAll = true)}
				>
					{m.Show_all_steps({ count: steps.data.length })}
				</Button>
			</td>
		</tr>
	{/if}
{:else}
	<tr
		class={['workflow-row-child bg-slate-50', density === 'comfortable' && 'text-xs']}
		style:--nest-pl={nestPl}
		style:--td-py={tdPy}
	>
		<td colspan={10} class="py-(--td-py) pr-(--td-px) pl-(--nest-pl) text-muted-foreground">
			{#if steps.isPending}
				<span class="inline-flex items-center gap-2">
					<Spinner size={12} />
					{m.Loading_steps()}
				</span>
			{:else if steps.isError}
				<span class="inline-flex items-center gap-1">
					{m.Steps_failed_to_load()}
					<IconButton
						icon={RefreshCw}
						variant="ghost"
						size="mini"
						tooltip={m.Retry()}
						aria-label={m.Retry()}
						onclick={() => steps.refetch()}
					/>
				</span>
			{:else}
				{m.No_steps_ran()}
			{/if}
		</td>
	</tr>
{/if}
