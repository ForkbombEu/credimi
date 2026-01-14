// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pause as PauseIcon, Play as PlayIcon, X as XIcon } from 'lucide-svelte';
import { toast } from 'svelte-sonner';
import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod';

import type { IconComponent } from '@/components/types';
import type { SchedulesResponse } from '@/pocketbase/types';

import { createForm } from '@/forms';
import { m } from '@/i18n';
import { pb } from '@/pocketbase';

import { scheduleModeSchema } from './schedule.utils';

//

const schedulePipelineFormSchema = z.object({
	pipeline_id: z.string(),
	schedule_mode: scheduleModeSchema
});

async function schedulePipeline(data: z.infer<typeof schedulePipelineFormSchema>) {
	if (data.schedule_mode.mode === 'monthly') {
		data.schedule_mode.day = data.schedule_mode.day - 1;
	}
	return pb.send('/api/my/schedules/start', {
		method: 'POST',
		body: data
	});
}

export function createSchedulePipelineForm(pipeline_id: string, onSuccess: () => void) {
	return createForm({
		adapter: zod(schedulePipelineFormSchema),
		onSubmit: async ({ form }) => {
			await schedulePipeline(form.data);
			toast.success(m.Pipeline_scheduled_successfully());
			// onSuccess();
		},
		initialData: {
			pipeline_id
		}
	});
}

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
		disabled: (schedule) => Boolean(schedule)
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
		disabled: (schedule) => Boolean(schedule)
	}
];
