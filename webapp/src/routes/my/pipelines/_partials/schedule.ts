// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { toast } from 'svelte-sonner';
import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod';

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

// export async function loadScheduledWorkflows(options = { fetch }): Promise<WorkflowSchedule[]> {
// 	const res = await pb.send<ListSchedulesResponse>('/api/my/schedules', {
// 		requestKey: null,
// 		fetch: options.fetch
// 	});
// 	return res.schedules ?? [];
// }

// type ListSchedulesResponse = {
// 	schedules?: WorkflowSchedule[];
// };

// export type WorkflowSchedule = {
// 	id: string;
// 	schedule_mode?: ScheduleMode;
// 	workflowType?: { name?: string };
// 	display_name: string;
// 	original_workflow_id: string;
// 	next_action_time?: string;
// 	paused?: boolean;
// };
