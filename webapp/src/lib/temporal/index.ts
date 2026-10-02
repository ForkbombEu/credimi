// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Canonical Temporal workflow execution statuses used in Credimi filters and badges. */
export const workflowStatuses = [
	'Running',
	'TimedOut',
	'Completed',
	'Failed',
	'ContinuedAsNew',
	'Canceled',
	'Terminated'
] as const;

export type WorkflowStatusType = (typeof workflowStatuses)[number];

export function isWorkflowStatus(status?: string | null | undefined): status is WorkflowStatusType {
	return (workflowStatuses as readonly string[]).includes(status ?? '');
}
