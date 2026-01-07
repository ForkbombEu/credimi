// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export interface FetchWorkflowsResponse {
	executions: Array<WorkflowExecutionWithChildren>;
}

export interface Failure {
	message?: string;
	source?: string;
	stackTrace?: string;
	cause?: Failure;
	failureInfo?: Record<string, string>;
}

export interface WorkflowExecutionWithChildren {
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
	children?: Array<WorkflowExecutionWithChildren>;
	failure?: Failure;
}
