// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import z from 'zod';

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

export type WorkflowExecution = z.infer<typeof workflowExecutionSchema>;

export const workflowsResponseSchema = z.object({
	executions: z.array(workflowExecutionSchema)
});
