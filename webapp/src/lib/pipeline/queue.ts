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

import { getPipelineRunner, getPipelineRunnerType, parsePipelineYaml } from './utils';

/** Runner-level status for a queued pipeline run (POST /api/pipeline/queue response) */
export type PipelineQueueRunnerStatus = {
	runner_id: string;
	status: PipelineQueueRunStatus;
	position: number;
	line_len: number;
	workflow_id?: string;
	run_id?: string;
	workflow_namespace?: string;
	error_message?: string;
};

/** Status values from the mobile runner semaphore workflow */
export type PipelineQueueRunStatus =
	| 'queued'
	| 'starting'
	| 'running'
	| 'failed'
	| 'canceled'
	| 'not_found';

/** Response body of POST /api/pipeline/queue (200 OK) */
export type PipelineQueueResponse = {
	ticket_id: string;
	enqueued_at?: string; // ISO 8601
	runner_ids?: string[];
	leader_runner_id?: string;
	required_runner_ids?: string[];
	status: PipelineQueueRunStatus;
	position: number;
	line_len: number;
	workflow_id?: string;
	run_id?: string;
	workflow_namespace?: string;
	error_message?: string;
	runners: PipelineQueueRunnerStatus[];
};

//

export async function enqueuePipeline(
	pipeline: PipelinesResponse
): Promise<Result<PipelineQueueResponse, string>> {
	try {
		const parsedYaml = parsePipelineYaml(pipeline.yaml);
		const runnerType = getPipelineRunnerType(pipeline);

		if (runnerType === 'global') {
			const runner = getPipelineRunner(pipeline.id);
			if (!runner) throw new Error('No runner found');
			if (parsedYaml.runtime) parsedYaml.runtime.global_runner_id = runner;
			else parsedYaml.runtime = { global_runner_id: runner };
		}

		const res = await pb.send<PipelineQueueResponse>('/api/pipeline/queue', {
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

export async function cancelQueueTicket(
	ticketId: string,
	runnerIds: string[]
): Promise<Result<PipelineQueueResponse, string>> {
	try {
		const params = new URLSearchParams();
		params.set('runner_ids', runnerIds.join(','));
		const url = `/api/pipeline/queue/${ticketId}?${params.toString()}`;
		const res = await pb.send<PipelineQueueResponse>(url, { method: 'DELETE' });
		if (res.status === 'canceled') {
			return ok(res);
		} else {
			return err(res.error_message ?? m.Failed_to_cancel_pipeline());
		}
	} catch (e) {
		return err(getExceptionMessage(e));
	}
}
