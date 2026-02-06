// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { MobileRunnersResponse } from '@/pocketbase/types';

export interface FetchWorkflowsResponse {
	executions: Array<WorkflowExecutionSummary>;
}

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
	status: string;
	displayName: string;
	results?: Array<{
		video: string;
		screenshot: string;
	}>;
	children?: Array<WorkflowExecutionSummary>;
	failure_reason?: string;
	global_runner_id?: string;
	runner_ids?: string[];
	runner_records?: Array<MobileRunnersResponse>;
	queue?: {
		ticket_id: string;
		position: number;
		line_len: number;
		runner_ids: string[];
	};
}
