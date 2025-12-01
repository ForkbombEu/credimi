// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { toast } from 'svelte-sonner';
import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod';

import { createForm } from '@/forms';
import { m } from '@/i18n';
import { pb } from '@/pocketbase';

//

export const SCHEDULING_INTERVALS = [
	// 'every_minute',
	'hourly',
	'daily',
	'weekly',
	'monthly'
] as const;

type SchedulingInterval = (typeof SCHEDULING_INTERVALS)[number];

type ScheduleWorkflowRequest = {
	workflowID: string;
	runID: string;
	interval: SchedulingInterval;
};

export async function scheduleWorkflow(data: ScheduleWorkflowRequest) {
	const res = await pb.send('/api/workflow/start-scheduled-workflow', {
		method: 'POST',
		body: {
			workflowID: data.workflowID,
			runID: data.runID,
			interval: data.interval
		}
	});
	console.log(res);
	return res;
}

export const schedulingIntervalLabels: Record<SchedulingInterval, string> = {
	// every_minute: m.every_minute(),
	hourly: m.hourly(),
	daily: m.daily(),
	weekly: m.weekly(),
	monthly: m.monthly()
};

export const scheduleWorkflowFormSchema = z.object({
	workflowID: z.string(),
	runID: z.string(),
	interval: z.enum(SCHEDULING_INTERVALS)
});

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
			workflowID,
			runID
		}
	});
}
