<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { runWithLoading } from '$lib/utils';
	import { EllipsisIcon, PauseIcon, PlayIcon, XIcon } from 'lucide-svelte';

	import type { IconComponent } from '@/components/types';
	import type { SchedulesResponse } from '@/pocketbase/types';

	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	type Props = {
		schedule: SchedulesResponse;
	};

	let { schedule }: Props = $props();

	//

	type ScheduleAction = {
		type: 'cancel' | 'pause' | 'resume';
		label: string;
		icon: IconComponent;
		action: (scheduleId: string, options?: { fetch: typeof fetch }) => Promise<void>;
		successMessage: string;
		disabled?: (schedule: SchedulesResponse | undefined) => boolean;
	};

	export const scheduleActions: ScheduleAction[] = [
		{
			type: 'cancel',
			label: m.Cancel(),
			icon: XIcon,
			successMessage: m.Schedule_cancelled_successfully(),
			action: async (scheduleId: string, options = { fetch }) => {
				return await pb.send(`/api/my/schedules/${scheduleId}/cancel`, {
					method: 'POST',
					requestKey: null,
					fetch: options.fetch
				});
			}
		},
		{
			type: 'pause',
			label: m.Pause(),
			icon: PauseIcon,
			successMessage: m.Schedule_paused_successfully(),
			action: async (scheduleId: string, options = { fetch }) => {
				return await pb.send(`/api/my/schedules/${scheduleId}/pause`, {
					method: 'POST',
					requestKey: null,
					fetch: options.fetch
				});
			},
			disabled: (schedule) => false
		},
		{
			type: 'resume',
			label: m.Resume(),
			icon: PlayIcon,
			successMessage: m.Schedule_resumed_successfully(),
			action: async (scheduleId: string, options = { fetch }) => {
				return await pb.send(`/api/my/schedules/${scheduleId}/resume`, {
					method: 'POST',
					requestKey: null,
					fetch: options.fetch
				});
			},
			disabled: (schedule) => false
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
				fn: () => action.action(schedule.temporal_schedule_id),
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
