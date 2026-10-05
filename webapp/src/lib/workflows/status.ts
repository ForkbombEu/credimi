// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { isWorkflowStatus, type WorkflowStatusType } from '$lib/temporal';

import type { WorkflowStatus } from './types';

const WORKFLOW_EXECUTION_STATUS_PREFIX = 'WorkflowExecutionStatus';

/**
 * Maps Temporal protojson status enums (e.g. WORKFLOW_EXECUTION_STATUS_RUNNING)
 * and already-readable labels into Credimi WorkflowStatus values.
 */
export function toReadableWorkflowStatus(raw: string | null | undefined): WorkflowStatus {
	if (!raw) return 'Unspecified';
	if (isWorkflowStatus(raw) || raw === 'Unspecified') return raw;

	const readable = fromScreamingEnum(raw, WORKFLOW_EXECUTION_STATUS_PREFIX);
	if (isWorkflowStatus(readable) || readable === 'Unspecified') {
		return readable as WorkflowStatus;
	}
	return 'Unspecified';
}

function fromScreamingEnum(value: string, prefix: string): string {
	const formatted = value
		.split('_')
		.map((word) => word.charAt(0) + word.slice(1).toLowerCase())
		.join('');
	return formatted.replace(prefix, '');
}

export type { WorkflowStatusType };
