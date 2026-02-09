// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowStatus } from '@forkbombeu/temporal-ui/dist/types/workflows';

//

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
	status: NonNullable<WorkflowStatus>;
	displayName: string;
	children?: Array<WorkflowExecutionSummary>;
	failure_reason?: string;
}
