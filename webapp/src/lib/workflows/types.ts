// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import z from 'zod/v3';

//

export const workflowExecutionInfoSchema = z
	.object({
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
		closeTime: z.string().optional(),
		executionDuration: z.string().optional(),
		historyLength: z.string().optional(),
		stateTransitionCount: z.string().optional(),
		status: z.string(),
		taskQueue: z.string(),
		type: z.object({
			name: z.string()
		})
	})
	.passthrough();

export type WorkflowExecutionInfo = z.infer<typeof workflowExecutionInfoSchema>;

export const workflowResponseSchema = z.object({
	workflowExecutionInfo: workflowExecutionInfoSchema,
	executionConfig: z.unknown(),
	pendingActivities: z.unknown()
});

export type WorkflowResponse = z.infer<typeof workflowResponseSchema>;
