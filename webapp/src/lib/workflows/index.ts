// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import z from 'zod';

export async function fetchUserWorkflows(fetchFn = fetch) {
	const data = await pb.send('/api/compliance/check/checks', {
		method: 'GET',
		fetch: fetchFn
	});
	return genericResponseSchema.safeParse(data);
}

const workflowExecutionSchema = z.object({
	execution: z.object({
		runId: z.string(),
		workflowId: z.string()
	}),
	executionTime: z.string(),
	memo: z.record(z.unknown()),
	rootExecution: z.object({
		runId: z.string(),
		workflowId: z.string()
	}),
	startTime: z.string(),
	endTime: z.string().optional(),
	status: z.string(),
	taskQueue: z.string(),
	type: z.object({
		name: z.string()
	})
});

const genericResponseSchema = z.object({
	executions: z.array(workflowExecutionSchema)
});
