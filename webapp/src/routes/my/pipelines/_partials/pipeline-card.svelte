<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';
	import type { MobileRunnersResponse } from '@/pocketbase/types';

	import { toWorkflowStatusReadable } from '@forkbombeu/temporal-ui';
	import { ArrowRightIcon, Cog, Pencil, PlayIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { userOrganization } from '$lib/app-state';
	import StatusCircle from '$lib/components/status-circle.svelte';
	import BlueButton from '$lib/layout/blue-button.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import { getPath } from '$lib/utils';
	import WorkflowsTableSmall from '$lib/workflows/workflows-table-small.svelte';
	import {
		runPipeline,
		getPipelineRunner,
		setPipelineRunner,
		getPipelineRunnerType
	} from '$lib/pipelines/utils';
	import SelectRunner from '$lib/pipelines/select-runner.svelte';

	import type { PocketbaseQueryResponse } from '@/pocketbase/query';

	import Button from '@/components/ui-custom/button.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import ScheduleActions from './schedule-actions.svelte';
	import SchedulePipelineForm from './schedule-pipeline-form.svelte';
	import { type EnrichedSchedule } from './types';

	//

	type Props = {
		pipeline: PocketbaseQueryResponse<'pipelines', ['schedules_via_pipeline', 'owner']>;
		workflows?: WorkflowExecutionSummary[];
	};

	let { pipeline = $bindable(), workflows }: Props = $props();

	//

	let runnerSelectionDialogOpen = $state(false);
	let runnerConfigDialogOpen = $state(false);

	const runnerType = $derived(getPipelineRunnerType(pipeline));
	const isRunnerSpecific = $derived(runnerType === 'specific');
	
	// Get the stored runner canonified path
	const storedRunnerPath = $derived(getPipelineRunner(pipeline.id));
	
	// For displaying the current runner name
	let currentRunnerName = $state<string | undefined>(undefined);
	
	// Fetch the runner name when storedRunnerPath changes
	$effect(() => {
		if (storedRunnerPath) {
			const filter = pb.filter('__canonified_path__ = {:path}', { path: storedRunnerPath });
			pb.collection('mobile_runners')
				.getFirstListItem(filter)
				.then((runner) => {
					currentRunnerName = runner.name;
				})
				.catch(() => {
					currentRunnerName = undefined;
				});
		} else {
			currentRunnerName = undefined;
		}
	});

	async function handleRunNow() {
		const runner = getPipelineRunner(pipeline.id);
		if (runner) {
			// Runner is already selected, run directly
			await runPipeline(pipeline, { global_runner_id: runner });
		} else {
			// No runner selected, open dialog
			runnerSelectionDialogOpen = true;
		}
	}

	function handleRunnerSelect(runner: MobileRunnersResponse) {
		const runnerPath = getPath(runner);
		setPipelineRunner(pipeline, runnerPath);
		runPipeline(pipeline, { global_runner_id: runnerPath });
		runnerSelectionDialogOpen = false;
	}

	function handleConfigRunnerSelect(runner: MobileRunnersResponse) {
		const runnerPath = getPath(runner);
		setPipelineRunner(pipeline, runnerPath);
		currentRunnerName = runner.name;
		runnerConfigDialogOpen = false;
	}

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
		<div class="flex gap-2">
			<Button onclick={handleRunNow}>
				<PlayIcon />{m.Run_now()}
			</Button>
			<IconButton
				icon={Cog}
				onclick={() => (runnerConfigDialogOpen = true)}
				disabled={isRunnerSpecific}
				tooltip={isRunnerSpecific ? m.Runner_configuration_not_available() : m.Configure_runner()}
			/>
		</div>
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

<!-- Runner Selection Dialog (for Run Now) -->
<Dialog
	bind:open={runnerSelectionDialogOpen}
	title={m.Select_runner()}
	description={m.Select_a_runner_to_execute_the_pipeline()}
>
	{#snippet content({ closeDialog })}
		<SelectRunner pipeline={pipeline} onRunnerSelect={handleRunnerSelect} />
	{/snippet}
</Dialog>

<!-- Runner Configuration Dialog -->
<Dialog
	bind:open={runnerConfigDialogOpen}
	title={m.Configure_runner()}
	description={m.Configure_the_runner_for_this_pipeline()}
>
	{#snippet content({ closeDialog })}
		{#if currentRunnerName}
			<div class="p-4">
				<T class="text-sm text-muted-foreground">{m.Current_runner()}: <span class="font-semibold">{currentRunnerName}</span></T>
			</div>
		{/if}
		<SelectRunner pipeline={pipeline} onRunnerSelect={handleConfigRunnerSelect} />
	{/snippet}
</Dialog>

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
