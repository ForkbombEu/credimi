<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import { Pencil, PlayIcon } from '@lucide/svelte';

	import type { PocketbaseQueryResponse } from '@/pocketbase/query';
	import type { OrganizationsResponse } from '@/pocketbase/types';

	import { RecordClone } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import ScheduleActions from './schedule-actions.svelte';
	import SchedulePipelineForm from './schedule-pipeline-form.svelte';
	import { scheduleModeLabel, type EnrichedSchedule, type ScheduleMode } from './types';
	import { runPipeline } from './utils';

	//

	type Props = {
		pipeline: PocketbaseQueryResponse<'pipelines', ['schedules_via_pipeline']>;
		organization: OrganizationsResponse;
	};

	let { pipeline = $bindable(), organization }: Props = $props();

	//

	let schedule = $derived.by(() => {
		const s = pipeline.expand?.schedules_via_pipeline?.find(
			(schedule) => schedule.owner === organization.id
		);
		return s as EnrichedSchedule | undefined;
	});

	const scheduleState = $derived.by(() => {
		if (schedule?.__schedule_status__.paused) return 'paused';
		else if (schedule?.__schedule_status__.paused === false) return 'active';
		else return 'not-scheduled';
	});
</script>

<DashboardCard
	record={pipeline}
	avatar={() => pb.files.getURL(organization, organization.logo)}
	path={[organization.canonified_name, pipeline.canonified_name]}
	badge={m.Yours()}
>
	{#snippet editAction()}
		<Button onclick={() => runPipeline(pipeline)}>
			<PlayIcon />{m.Run_now()}
		</Button>
		<!-- <Button
		href="/my/pipelines/settings-{pipeline.id}"
		variant="outline"
		size="icon"
	>
		<CogIcon />
	</Button> -->
		<RecordClone collectionName="pipelines" recordId={pipeline.id} size="md" />
		<IconButton href="/my/pipelines/edit-{pipeline.id}" icon={Pencil} />
	{/snippet}

	{#snippet content()}
		<div class="flex justify-between">
			<div class="flex flex-wrap items-center gap-1.5 text-sm">
				{@render circle({
					'bg-green-500': scheduleState === 'active',
					'bg-yellow-500': scheduleState === 'paused',
					'bg-gray-50 border': scheduleState === 'not-scheduled'
				})}
				<T>
					{#if schedule}
						{#if scheduleState === 'paused'}
							<span class="font-bold">{m.Scheduling_paused()}</span>
						{:else if scheduleState === 'active'}
							<span class="font-bold">{m.scheduled()}:</span>
						{/if}
						<span class={[scheduleState === 'paused' && 'pl-1 opacity-30']}>
							{scheduleModeLabel(schedule.mode as ScheduleMode)}
						</span>
					{:else}
						{m.Pipeline_execution_is_not_scheduled()}
					{/if}
				</T>
				{#if schedule && scheduleState === 'active'}
					<T>
						<span class="font-bold">{m.next_run()}:</span>
						{schedule.__schedule_status__.next_action_time}
					</T>
				{/if}
			</div>

			<div class="-m-2">
				{#if !schedule}
					<SchedulePipelineForm {pipeline} />
				{:else}
					<ScheduleActions bind:schedule />
				{/if}
			</div>
		</div>
	{/snippet}
</DashboardCard>

{#snippet circle(className: ClassValue)}
	<div class={['size-2 rounded-full', className]}></div>
{/snippet}
