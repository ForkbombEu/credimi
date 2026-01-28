<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';

	import { toWorkflowStatusReadable } from '@forkbombeu/temporal-ui';
	import { ArrowRightIcon, Pencil, PlayIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { userOrganization } from '$lib/app-state';
	import { getPath } from '$lib/utils';
	import StatusCircle from '$lib/components/status-circle.svelte';
	import BlueButton from '$lib/layout/blue-button.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import WorkflowsTableSmall from '$lib/workflows/workflows-table-small.svelte';

	import type { PocketbaseQueryResponse } from '@/pocketbase/query';

	import Button from '@/components/ui-custom/button.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import SelectRunner from '@/components/select-runner.svelte';
	import { SelectRunnerForm } from '@/components/select-runner.svelte.js';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import ScheduleActions from './schedule-actions.svelte';
	import SchedulePipelineForm from './schedule-pipeline-form.svelte';
	import { type EnrichedSchedule } from './types';
	import { 
		runPipeline, 
		pipelineRequiresGlobalRunner,
		getStoredRunner,
		storeRunner 
	} from './utils';

	//

	type Props = {
		pipeline: PocketbaseQueryResponse<'pipelines', ['schedules_via_pipeline', 'owner']>;
		workflows?: WorkflowExecutionSummary[];
	};

	let { pipeline = $bindable(), workflows }: Props = $props();

	//

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

	const avatar = $derived.by(() => {
		const owner = pipeline.expand?.owner;
		if (!owner) return undefined;
		return pb.files.getURL(owner, owner.logo);
	});

	const hasWorkflows = $derived(workflows && workflows.length > 0);

	const isPublic = $derived(pipeline.owner !== userOrganization.current?.id);

	// Runner selection state
	let showRunnerModal = $state(false);
	let runnerForm = $state(new SelectRunnerForm());
	let isRunning = $state(false);

	async function handleRunPipeline() {
		// Prevent multiple simultaneous executions
		if (isRunning) return;
		isRunning = true;

		try {
			// Check if pipeline requires global runner
			const requiresRunner = pipelineRequiresGlobalRunner(pipeline.yaml);
			
			if (!requiresRunner) {
				// Run directly without runner selection
				await runPipeline(pipeline);
				return;
			}

			// Check for stored runner
			const storedRunner = getStoredRunner(pipeline.id);
			if (storedRunner) {
				// Use stored runner
				await runPipeline(pipeline, storedRunner);
				return;
			}

			// Show modal to select runner
			runnerForm = new SelectRunnerForm((runner) => {
				const runnerPath = getPath(runner);
				storeRunner(pipeline.id, runnerPath);
				showRunnerModal = false;
				runPipeline(pipeline, runnerPath);
			});
			showRunnerModal = true;
		} finally {
			isRunning = false;
		}
	}
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
		<Button onclick={handleRunPipeline}>
			<PlayIcon />{m.Run_now()}
		</Button>
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

<!-- Runner Selection Modal -->
<Dialog bind:open={showRunnerModal} title={m.Select_runner()}>
	{#snippet content()}
		<SelectRunner form={runnerForm} showSelected={false} />
	{/snippet}
</Dialog>

{#snippet editAction()}
	<IconButton href="/my/pipelines/edit-{pipeline.id}" icon={Pencil} tooltip={m.Edit()} />
{/snippet}

{#snippet content()}
	{#if workflows && workflows.length > 0}
		<div class="space-y-3">
			<div class="flex items-center justify-between gap-1">
				<T class="text-sm font-medium">{m.Recent_workflows()}</T>
				<BlueButton
					compact
					href={resolve('/my/pipelines/[pipeline_id]', { pipeline_id: pipeline.id })}
				>
					{m.view_all()}
					<ArrowRightIcon />
				</BlueButton>
			</div>

			<WorkflowsTableSmall {workflows} />
		</div>
	{/if}
{/snippet}
