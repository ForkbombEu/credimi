// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowStatusType } from '$lib/temporal';

import z from 'zod/v3';

//

/** Fields Credimi reads from Temporal describe (protojson) via /api/my/workflows/... */
export const workflowExecutionInfoSchema = z
	.object({
		execution: z.object({
			runId: z.string(),
			workflowId: z.string()
		}),
		type: z.object({
			name: z.string()
		}),
		status: z.string(),
		startTime: z.string().optional(),
		closeTime: z.string().optional(),
		memo: z.unknown().optional()
	})
	.passthrough();

export type WorkflowExecutionInfo = z.infer<typeof workflowExecutionInfoSchema>;

export const workflowResponseSchema = z
	.object({
		workflowExecutionInfo: workflowExecutionInfoSchema,
		failure_reason: z.string().optional(),
		devices: z
			.array(
				z.object({
					device_id: z.string(),
					name: z.string(),
					live_view: z.boolean()
				})
			)
			.optional()
	})
	.passthrough();

export type WorkflowResponse = z.infer<typeof workflowResponseSchema>;

/** Readable workflow status labels (plus Unspecified for unknown/empty). */
export type WorkflowStatus = WorkflowStatusType | 'Unspecified';
