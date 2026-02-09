// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { runWithLoading } from '$lib/utils';
import { toast } from 'svelte-sonner';

import type { PipelinesResponse } from '@/pocketbase/types';

import { m } from '@/i18n';

import * as PipelineQueue from './queue';

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
				position: result.value.position ?? 0,
				line_len: result.value.line_len ?? 0
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
