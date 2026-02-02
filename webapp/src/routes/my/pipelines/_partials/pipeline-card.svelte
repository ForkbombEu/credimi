<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	/* eslint-disable perfectionist/sort-imports */
	import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';

	import { resolve } from '$app/paths';
	import { userOrganization } from '$lib/app-state';
	import StatusCircle from '$lib/components/status-circle.svelte';
	import BlueButton from '$lib/layout/blue-button.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import RunnerSelectModal from '$lib/pipeline/runner-select-modal.svelte';
	import { getPipelineRunner, getPipelineRunnerType, runPipeline } from '$lib/pipeline/utils';
	import { getPath } from '$lib/utils';
	import WorkflowsTableSmall from '$lib/workflows/workflows-table-small.svelte';
	import { toWorkflowStatusReadable } from '@forkbombeu/temporal-ui';
	import { ArrowRightIcon, Cog, Pencil, PlayIcon } from '@lucide/svelte';

	import type { PocketbaseQueryResponse } from '@/pocketbase/query';

	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import * as ButtonGroup from '@/components/ui/button-group';
	import ScheduleActions from './schedule-actions.svelte';
	import SchedulePipelineForm from './schedule-pipeline-form.svelte';
	import { type EnrichedSchedule } from './types';

	//

	type Props = {
		pipeline: PocketbaseQueryResponse<'pipelines', ['schedules_via_pipeline', 'owner']>;
		workflows?: WorkflowExecutionSummary[];
	};

	let { pipeline = $bindable(), workflows }: Props = $props();

	// Running

	let runnerSelectionDialogOpen = $state(false);
	let runPipelineAfterRunnerSelect = $state(false);

	async function handleRunNow() {
		const runner = getPipelineRunner(pipeline.id);
		if (runner) {
			await runPipeline(pipeline, { global_runner_id: runner });
			runPipelineAfterRunnerSelect = false;
		} else {
			runPipelineAfterRunnerSelect = true;
			runnerSelectionDialogOpen = true;
		}
	}

	// Scheduling

	let schedule = $derived.by(() => {
		const s = pipeline.expand?.schedules_via_pipeline?.find(
			(schedule) => schedule.owner === userOrganization.current?.id
		);
		return s as EnrichedSchedule | undefined;
	});

	const isRunning = $derived(
		workflows?.some((workflow) => {
			const status = toWorkflowStatusReadable(workflow.status);
			return status === 'Running';
		})
	);

	// Flags for displaying UI elements

	const avatar = $derived.by(() => {
		const owner = pipeline.expand?.owner;
		if (!owner) return undefined;
		return pb.files.getURL(owner, owner.logo);
	});

	const hasWorkflows = $derived(workflows && workflows.length > 0);

	const isPublic = $derived(pipeline.owner !== userOrganization.current?.id);

	const runnerType = $derived(getPipelineRunnerType(pipeline));
	const isRunnerSpecific = $derived(runnerType === 'specific');
	$inspect(isRunnerSpecific);
</script>

<DashboardCard
	record={pipeline}
	{avatar}
	badge={isPublic ? m.Public() : undefined}
	content={hasWorkflows ? content : undefined}
	editAction={isPublic ? undefined : editAction}
	hideActions={isPublic ? ['delete', 'edit', 'publish'] : undefined}
>
	{#snippet nameRight()}
		{#if isRunning}
			<Badge
				variant="secondary"
				class="flex items-center gap-1.5 bg-green-100 text-green-800"
			>
				<StatusCircle size={12} />
				{m.Running()}
			</Badge>
		{/if}
	{/snippet}

	{#snippet actions()}
		<ButtonGroup.Root>
			<Button onclick={handleRunNow}>
				<PlayIcon />{m.Run_now()}
			</Button>
			<IconButton
				icon={Cog}
				variant="default"
				class="rounded-none rounded-r-md border-l border-l-slate-500"
				onclick={() => (runnerSelectionDialogOpen = true)}
				disabled={isRunnerSpecific}
				tooltip={isRunnerSpecific
					? m.Runner_configuration_not_available()
					: m.Configure_runner()}
			/>
		</ButtonGroup.Root>

		{#if !schedule}
			<SchedulePipelineForm {pipeline} />
		{:else}
			<ScheduleActions
				bind:schedule
				onCancel={() => {
					schedule = undefined;
				}}
			/>
		{/if}
	{/snippet}
</DashboardCard>

<RunnerSelectModal
	{pipeline}
	bind:open={runnerSelectionDialogOpen}
	onSelect={() => {
		if (!runPipelineAfterRunnerSelect) return;
		handleRunNow();
	}}
/>

{#snippet editAction()}
	<IconButton
		href={resolve('/my/pipelines/(group)/[...path]/edit', { path: getPath(pipeline, true) })}
		icon={Pencil}
		tooltip={m.Edit()}
	/>
{/snippet}

{#snippet content()}
	{#if workflows && workflows.length > 0}
		<div class="space-y-3">
			<div class="flex items-center justify-between gap-1">
				<T class="text-sm font-medium">{m.Recent_workflows()}</T>
				<BlueButton
					compact
					href={resolve('/my/pipelines/[...pipeline_path]', {
						pipeline_path: getPath(pipeline, true)
					})}
				>
					{m.view_all()}
					<ArrowRightIcon />
				</BlueButton>
			</div>

			<WorkflowsTableSmall {workflows} />
		</div>
	{/if}
{/snippet}
