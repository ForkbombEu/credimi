// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath, runWithLoading } from '$lib/utils';
import { toast } from 'svelte-sonner';

import type { PipelinesResponse } from '@/pocketbase/types';

import { goto, m } from '@/i18n';
import { pb } from '@/pocketbase';

//

export async function runPipeline(pipeline: PipelinesResponse) {
	const result = await runWithLoading({
		fn: async () => {
			return await pb.send('/api/pipeline/start', {
				method: 'POST',
				body: {
					yaml: pipeline.yaml,
					pipeline_identifier: getPath(pipeline)
				}
			});
		},
		showSuccessToast: false
	});

	if (result?.result) {
		const { workflowId, workflowRunId } = result.result;
		const workflowUrl =
			workflowId && workflowRunId
				? `/my/tests/runs/${workflowId}/${workflowRunId}`
				: undefined;

		toast.success(m.Pipeline_started_successfully(), {
			description: m.View_workflow_details(),
			duration: 10000,
			...(workflowUrl && {
				action: {
					label: m.View(),
					onClick: () => goto(workflowUrl)
				}
			})
		});
	}
}
