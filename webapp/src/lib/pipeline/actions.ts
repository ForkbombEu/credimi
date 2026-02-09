// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { XIcon } from '@lucide/svelte';
import { Workflow } from '$lib';
import { runWithLoading } from '$lib/utils';
import { toast } from 'svelte-sonner';
import { err, ok } from 'true-myth/result';

import type { DropdownMenuItem } from '@/components/ui-custom/dropdown-menu.svelte';
import type { PipelinesResponse } from '@/pocketbase/types';

import { m } from '@/i18n';
import { getExceptionMessage } from '@/utils/errors';

import type { ExecutionSummary } from './workflows';

import * as PipelineQueue from './queue';
import * as PipelineWorkflows from './workflows';

//

export async function run(pipeline: PipelinesResponse) {
	const result = await runWithLoading({
		fn: () => PipelineQueue.enqueue(pipeline),
		showSuccessToast: false
	});

	if (!result) {
		toast.error('Unexpected error');
		return;
	}

	if (result.isErr) {
		toast.error(result.error);
		return;
	}

	const runnerIds = result.value.runner_ids ?? [];
	if (!result.value.ticket_id || runnerIds.length === 0) {
		toast.error(m.Failed_to_enqueue_pipeline());
		return;
	}

	if (result.value.status === 'failed') {
		toast.error(result.value.error_message ?? m.Failed_to_enqueue_pipeline());
		return;
	}

	if (result.value.status === 'queued') {
		toast.info(
			m.Pipeline_queued({
				position: result.value.position ?? 0 + 1
			}),
			{
				duration: 5000,
				action: {
					label: m.Cancel(),
					onClick: async () => {
						if (!result.value.ticket_id) {
							toast.error(m.Failed_to_cancel_pipeline());
							return;
						}
						const response = await PipelineQueue.cancel(
							result.value.ticket_id,
							runnerIds
						);
						if (response.isOk) toast.success(m.Pipeline_execution_canceled());
						else toast.error(response.error);
					}
				}
			}
		);
	}
}

export async function cancel(workflow: PipelineWorkflows.ExecutionSummary) {
	const { execution, queue } = workflow;
	const result = await runWithLoading({
		fn: async () => {
			try {
				if (queue) {
					return await PipelineQueue.cancel(queue.ticket_id, queue.runner_ids);
				} else if (workflow.status === 'Running') {
					const res = await Workflow.cancel(execution.workflowId, execution.runId);
					return ok(res);
				} else {
					return err(m.Failed_to_cancel_pipeline());
				}
			} catch (e) {
				return err(getExceptionMessage(e));
			}
		},
		showSuccessToast: false
	});

	if (!result) {
		toast.error('Unexpected error');
		return;
	}

	if (result.isErr) {
		toast.error(result.error);
		return;
	}

	if (result.isOk) {
		toast.success(m.Pipeline_execution_canceled());
	}
}

//

export function makeDropdownActions(workflow: ExecutionSummary): DropdownMenuItem[] {
	return [
		{
			label: m.Cancel(),
			icon: XIcon,
			onclick: () => cancel(workflow),
			disabled: workflow.status !== 'Running' || !workflow.queue
		}
	];
}
