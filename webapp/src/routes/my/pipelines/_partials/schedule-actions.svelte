<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CalendarIcon, PauseIcon, PlayIcon, XIcon } from '@lucide/svelte';
	import { runWithLoading } from '$lib/utils';

	import type { IconComponent } from '@/components/types';

	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import {
		getScheduleState,
		scheduleModeLabel,
		type EnrichedSchedule,
		type ScheduleMode
	} from './types';

	//

	type Props = {
		schedule: EnrichedSchedule;
		onCancel?: () => void;
	};

	let { schedule = $bindable(), onCancel }: Props = $props();

	//

	const scheduleState = $derived(getScheduleState(schedule));

	type ScheduleAction = {
		type: 'cancel' | 'pause' | 'resume';
		label: string;
		icon: IconComponent;
		action: (schedule: EnrichedSchedule, options?: { fetch: typeof fetch }) => Promise<void>;
		successMessage: string;
		disabled?: (schedule: EnrichedSchedule | undefined) => boolean;
	};

	export const scheduleActions: ScheduleAction[] = [
		{
			type: 'cancel',
			label: m.Cancel(),
			icon: XIcon,
			successMessage: m.Schedule_cancelled_successfully(),
			action: async (schedule, options = { fetch }) => {
				const id = schedule.temporal_schedule_id;
				await pb.send(`/api/my/schedules/${id}/cancel`, {
					method: 'POST',
					requestKey: null,
					fetch: options.fetch
				});
				onCancel?.();
			}
		},
		{
			type: 'pause',
			label: m.Pause(),
			icon: PauseIcon,
			successMessage: m.Schedule_paused_successfully(),
			action: async (schedule, options = { fetch }) => {
				const id = schedule.temporal_schedule_id;
				await pb.send(`/api/my/schedules/${id}/pause`, {
					method: 'POST',
					requestKey: null,
					fetch: options.fetch
				});
				schedule.__schedule_status__.paused = true;
			},
			disabled: (schedule) => Boolean(schedule?.__schedule_status__.paused)
		},
		{
			type: 'resume',
			label: m.Resume(),
			icon: PlayIcon,
			successMessage: m.Schedule_resumed_successfully(),
			action: async (schedule, options = { fetch }) => {
				const id = schedule.temporal_schedule_id;
				await pb.send(`/api/my/schedules/${id}/resume`, {
					method: 'POST',
					requestKey: null,
					fetch: options.fetch
				});
				schedule.__schedule_status__.paused = false;
			},
			disabled: (schedule) => !schedule?.__schedule_status__.paused
		}
	];
</script>

<DropdownMenu
	title={m.Manage_scheduling()}
	items={scheduleActions.map((action) => ({
		label: action.label,
		icon: action.icon,
		disabled: !action.disabled ? false : action.disabled(schedule),
		onclick: () =>
			runWithLoading({
				fn: () => action.action(schedule),
				successText: action.successMessage,
				showSuccessToast: true
			})
	}))}
>
	{#snippet trigger({ props })}
		<IconButton
			{...props}
			class={[props.class, 'overflow-hidden']}
			icon={CalendarIcon}
			tooltip={m.Manage_scheduling()}
		>
			{#if scheduleState !== 'not-scheduled'}
				<div
					class={[
						'absolute right-0 bottom-0 left-0 h-[3px] ',
						{
							'bg-green-500': scheduleState === 'active',
							'bg-yellow-500': scheduleState === 'paused'
						}
					]}
				></div>
			{/if}
		</IconButton>
	{/snippet}

	{#snippet subtitle()}
		<div class="space-y-1 px-2 pb-1 text-xs text-slate-600">
			<T>
				<span class="font-medium">{m.interval()}</span><br />
				{scheduleModeLabel(schedule.mode as ScheduleMode)}
			</T>

			{#if scheduleState === 'active'}
				<T>
					<span class="font-medium">{m.next_run()}:</span><br />
					{schedule.__schedule_status__.next_action_time}
				</T>
			{/if}
		</div>
	{/snippet}
</DropdownMenu>
<!-- 
<div class="flex justify-between">
	<div class="flex flex-wrap items-center gap-1.5 text-sm">
		<ScheduleState state={scheduleState} />
		{#if schedule && scheduleState === 'active'}
		
			<T class="text-slate-300">|</T>
			<T>
				<span class="font-bold">{m.next_run()}:</span>
				{schedule.__schedule_status__.next_action_time}
			</T>
		{/if}
	</div>
</div> -->
