// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowExecution } from '$lib/credimiClient.generated';
import { Array } from 'effect';

//

export type WorkflowWithChildren = WorkflowExecution & {
	children?: WorkflowExecution[];
};

export function groupWorkflowsWithChildren(workflows: WorkflowExecution[]): WorkflowWithChildren[] {
	const [withoutParent, withParent] = Array.partition(workflows, (w) => Boolean(w.parent));

	return withoutParent.map((w) => ({
		...w,
		children: withParent.filter((c) => c.parent?.workflowId === w.id)
	}));
}
