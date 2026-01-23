<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowExecutionSummary } from '$lib/workflows/queries.types';

	import { toWorkflowStatusReadable } from '@forkbombeu/temporal-ui';
	import { ArrowRightIcon, Pencil, PlayIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import StatusCircle from '$lib/components/status-circle.svelte';
	import BlueButton from '$lib/layout/blue-button.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import WorkflowsTableSmall from '$lib/workflows/workflows-table-small.svelte';

	import type { PocketbaseQueryResponse } from '@/pocketbase/query';
	import type { OrganizationsResponse } from '@/pocketbase/types';

	import { RecordClone } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import ScheduleActions from './schedule-actions.svelte';
	import SchedulePipelineForm from './schedule-pipeline-form.svelte';
	import ScheduleState from './schedule-state-display.svelte';
	import {
		getScheduleState,
		scheduleModeLabel,
		type EnrichedSchedule,
		type ScheduleMode
	} from './types';
	import { runPipeline } from './utils';

	//

	type Props = {
		pipeline: PocketbaseQueryResponse<'pipelines', ['schedules_via_pipeline']>;
		organization: OrganizationsResponse;
		workflows?: WorkflowExecutionSummary[];
	};

	let { pipeline = $bindable(), organization, workflows }: Props = $props();

	//

	let schedule = $derived.by(() => {
		const s = pipeline.expand?.schedules_via_pipeline?.find(
			(schedule) => schedule.owner === organization.id
		);
		return s as EnrichedSchedule | undefined;
	});

	const scheduleState = $derived(getScheduleState(schedule));

	const isRunning = $derived(
		workflows?.some((workflow) => {
			const status = toWorkflowStatusReadable(workflow.status);
			return status === 'Running';
		})
	);
</script>

<DashboardCard
	record={pipeline}
	avatar={() => pb.files.getURL(organization, organization.logo)}
	path={[organization.canonified_name, pipeline.canonified_name]}
	badge={m.Yours()}
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

	{#snippet editAction()}
		<Button onclick={() => runPipeline(pipeline)}>
			<PlayIcon />{m.Run_now()}
		</Button>
		<RecordClone collectionName="pipelines" recordId={pipeline.id} size="md" />
		<IconButton href="/my/pipelines/edit-{pipeline.id}" icon={Pencil} />
	{/snippet}

	{#snippet content()}
		<div class="flex justify-between">
			<div class="flex flex-wrap items-center gap-1.5 text-sm">
				<ScheduleState state={scheduleState} />
				{#if schedule && scheduleState === 'active'}
					<T>
						{scheduleModeLabel(schedule.mode as ScheduleMode)}
					</T>
					<T class="text-slate-300">|</T>
					<T>
						<span class="font-bold">{m.next_run()}:</span>
						{schedule.__schedule_status__.next_action_time}
					</T>
				{/if}
			</div>

			<div>
				{#if !schedule}
					<SchedulePipelineForm {pipeline} />
				{:else}
					<ScheduleActions bind:schedule />
				{/if}
			</div>
		</div>

		{#if workflows && workflows.length > 0}
			<Separator />

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
</DashboardCard>
