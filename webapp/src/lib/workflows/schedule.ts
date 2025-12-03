// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pause as PauseIcon, Play as PlayIcon, X as XIcon } from 'lucide-svelte';
import { toast } from 'svelte-sonner';
import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod';

import type { IconComponent } from '@/components/types';
import type { SelectOption } from '@/components/ui-custom/utils';

import { createForm } from '@/forms';
import { m } from '@/i18n';
import { pb } from '@/pocketbase';

//

const scheduleModeSchema = z.union([
	z.object({
		mode: z.literal('daily')
	}),
	z.object({
		mode: z.literal('weekly'),
		day: z.number().min(0).max(6)
	}),
	z.object({
		mode: z.literal('monthly'),
		day: z.number().min(1).max(31)
	})
]);

type ScheduleMode = z.infer<typeof scheduleModeSchema>;
type ScheduleModeName = ScheduleMode['mode'];

//

export const scheduleWorkflowFormSchema = z.object({
	workflow_id: z.string(),
	run_id: z.string(),
	schedule_mode: scheduleModeSchema
});

export async function scheduleWorkflow(data: z.infer<typeof scheduleWorkflowFormSchema>) {
	if (data.schedule_mode.mode === 'monthly') {
		data.schedule_mode.day = data.schedule_mode.day - 1;
	}
	return pb.send('/api/my/schedules/start', {
		method: 'POST',
		body: data
	});
}

export function createScheduleWorkflowForm(props: {
	workflowID: string;
	runID: string;
	onSuccess: () => void;
}) {
	const { workflowID, runID, onSuccess } = props;
	return createForm({
		adapter: zod(scheduleWorkflowFormSchema),
		onSubmit: async ({ form }) => {
			await scheduleWorkflow(form.data);
			toast.success(m.Workflow_scheduled_successfully());
			onSuccess();
		},
		initialData: {
			workflow_id: workflowID,
			run_id: runID
		}
	});
}

//

export const scheduleModeOptions: SelectOption<ScheduleModeName>[] = [
	{
		label: m.daily(),
		value: 'daily'
	},
	{
		label: m.weekly(),
		value: 'weekly'
	},
	{
		label: m.monthly(),
		value: 'monthly'
	}
];

//

export async function loadScheduledWorkflows(options = { fetch }) {
	const res = await pb.send('/api/my/schedules', {
		requestKey: null,
		fetch: options.fetch
	});
	return res as ListSchedulesResponse;
}

type ListSchedulesResponse = {
	schedules?: WorkflowSchedule[];
};

export type WorkflowSchedule = {
	id: string;
	schedule_mode?: ScheduleMode;
	workflowType?: { name?: string };
	display_name: string;
	original_workflow_id: string;
	paused?: boolean;
};

//

type ScheduleAction = {
	type: 'cancel' | 'pause' | 'resume';
	label: string;
	icon: IconComponent;
	action: (scheduleId: string, options?: { fetch: typeof fetch }) => Promise<void>;
	successMessage: string;
	disabled?: (schedule: WorkflowSchedule) => boolean;
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
		disabled: (schedule) => Boolean(schedule.paused)
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
		disabled: (schedule) => !schedule.paused
	}
];
