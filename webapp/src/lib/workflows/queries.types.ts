// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowStatus } from './types';

//

export interface FetchWorkflowsResponse {
	executions: Array<WorkflowExecutionSummary>;
}

/** Queue metadata emitted on queued pipeline summaries (`WorkflowQueueSummary` in Go). */
export interface WorkflowQueueSummary {
	ticket_id: string;
	position: number;
	line_len: number;
	device_ids?: string[];
}

/** Mobile/pipeline artifact URLs (`pipelineresults.PipelineResults` in Go). */
export interface WorkflowExecutionResult {
	video: string;
	screenshot: string;
	log: string;
}

/**
 * Shared JSON shape for workflow execution summaries returned by Credimi list
 * endpoints. Mirrors `WorkflowExecutionSummary` in
 * `pkg/internal/apis/handlers/shared.go` without requiring a 1:1 identical type
 * on every consumer — pipeline UI extends this via `ExecutionSummary`.
 */
export interface WorkflowExecutionSummary {
	execution: {
		workflowId: string;
		runId: string;
	};
	type: {
		name: string;
	};
	startTime: string;
	endTime?: string;
	duration?: string;
	enqueuedAt?: string;
	status: WorkflowStatus;
	displayName: string;
	queue?: WorkflowQueueSummary;
	children?: Array<WorkflowExecutionSummary>;
	results?: Array<WorkflowExecutionResult>;
	maestro_screenshots?: string[];
	report?: string;
	fcaf_report?: string;
	fcaf_report_pdf?: string;
	failure_reason?: string;
	has_logs?: boolean;
}
