// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';
import { err, ok, Result } from 'true-myth/result';
import { stringify } from 'yaml';

import type { PipelinesResponse } from '@/pocketbase/types';

import { m } from '@/i18n';
import { pb } from '@/pocketbase';
import { getExceptionMessage } from '@/utils/errors';

import * as PipelineRunner from './runner';
import { parseYaml } from './utils';

//

export type Status = 'queued' | 'starting' | 'running' | 'failed' | 'canceled' | 'not_found';

/** Shared optional fields that the API may include on any response */
type APIResponseBase = {
	ticket_id?: string;
	enqueued_at?: string;
	runner_ids?: string[];
	position?: number;
	line_len?: number;
	workflow_id?: string;
	run_id?: string;
	error_message?: string;
};

/**
 * Discriminated union for POST /api/pipeline/queue, GET /api/pipeline/queue/{ticket}, DELETE /api/pipeline/queue/{ticket}.
 * Narrow by `status` to get the appropriate shape.
 */
export type APIResponse =
	| (APIResponseBase & { status: 'queued' | 'starting' })
	| (APIResponseBase & { status: 'running'; workflow_id: string; run_id: string })
	| (APIResponseBase & { status: 'failed' | 'canceled' })
	| (APIResponseBase & { status: 'not_found' });

export async function enqueue(pipeline: PipelinesResponse): Promise<Result<APIResponse, string>> {
	try {
		const parsedYaml = parseYaml(pipeline.yaml);
		const runnerType = PipelineRunner.getType(pipeline);

		if (runnerType === 'global') {
			const runner = PipelineRunner.get(pipeline.id);
			if (!runner) throw new Error('No runner found');
			if (parsedYaml.runtime) parsedYaml.runtime.global_runner_id = runner;
			else parsedYaml.runtime = { global_runner_id: runner };
		}

		const res = await pb.send<APIResponse>('/api/pipeline/queue', {
			method: 'POST',
			body: {
				pipeline_identifier: getPath(pipeline),
				yaml: stringify(parsedYaml)
			}
		});
		if (res.status === 'queued' || res.status === 'starting' || res.status === 'running') {
			return ok(res);
		} else {
			return err(res.error_message ?? m.Failed_to_enqueue_pipeline());
		}
	} catch (e) {
		return err(getExceptionMessage(e));
	}
}

export async function cancel(
	ticketId: string,
	runnerIds: string[]
): Promise<Result<APIResponse, string>> {
	try {
		const params = new URLSearchParams();
		params.set('runner_ids', runnerIds.join(','));
		const url = `/api/pipeline/queue/${ticketId}?${params.toString()}`;
		const res = await pb.send<APIResponse>(url, { method: 'DELETE' });
		if (res.status === 'canceled') {
			return ok(res);
		} else {
			return err(res.error_message ?? m.Failed_to_cancel_pipeline());
		}
	} catch (e) {
		return err(getExceptionMessage(e));
	}
}
