<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pencil, PlayIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';

	import type { PocketbaseQueryResponse } from '@/pocketbase/query';
	import type { OrganizationsResponse } from '@/pocketbase/types';

	import { RecordClone } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
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
	};

	let { pipeline = $bindable(), organization }: Props = $props();

	//

	let schedule = $derived.by(() => {
		const s = pipeline.expand?.schedules_via_pipeline?.find(
			(schedule) => schedule.owner === organization.id
		);
		return s as EnrichedSchedule | undefined;
	});

	const scheduleState = $derived(getScheduleState(schedule));
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

			<div class="-m-2">
				{#if !schedule}
					<SchedulePipelineForm {pipeline} />
				{:else}
					<ScheduleActions bind:schedule />
				{/if}
			</div>
		</div>

		<Separator />

		<Button
			variant="outline"
			href={resolve('/my/pipelines/[pipeline_id]', { pipeline_id: pipeline.id })}
		>
			{m.view_details()}
		</Button>
	{/snippet}
</DashboardCard>

<!-- <Button
		href="/my/pipelines/settings-{pipeline.id}"
		variant="outline"
		size="icon"
	>
		<CogIcon />
	</Button> -->
