<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { runWithLoading } from '$lib/utils';
	import { EllipsisIcon, PauseIcon, PlayIcon, XIcon } from '@lucide/svelte';

	import type { IconComponent } from '@/components/types';

	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import type { EnrichedSchedule } from './types';

	//

	type Props = {
		schedule: EnrichedSchedule;
		onCancel?: () => void;
	};

	let { schedule = $bindable(), onCancel }: Props = $props();

	//

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
	buttonVariants={{ size: 'sm', variant: 'ghost', class: 'text-blue-600 hover:text-blue-600' }}
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
	{#snippet trigger()}
		<EllipsisIcon />
		{m.Manage()}
	{/snippet}
</DropdownMenu>
