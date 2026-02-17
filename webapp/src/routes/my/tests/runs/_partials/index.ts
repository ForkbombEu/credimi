// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pipeline, Workflow } from '$lib';
import { isWorkflowStatus, workflowStatuses } from '$lib/temporal';

import { m } from '@/i18n';

//

const ExtendedStatusTag = Pipeline.Workflows.StatusTag;
export { ExtendedStatusTag };

//

type ExtendedWorkflowStatus = Workflow.WorkflowStatus | Pipeline.Workflows.Status;

export const ALL_WORKFLOW_STATUSES: ExtendedWorkflowStatus[] = [
	...workflowStatuses.filter((status) => status !== null),
	Pipeline.Workflows.QUEUED_STATUS
];

export function isExtendedWorkflowStatus(status?: string | null): status is ExtendedWorkflowStatus {
	return isWorkflowStatus(status) || status === Pipeline.Workflows.QUEUED_STATUS;
}

export function getStatusQueryParam(url: URL): ExtendedWorkflowStatus | undefined {
	const status = url.searchParams.get(Workflow.WORKFLOW_STATUS_QUERY_PARAM);
	if (isExtendedWorkflowStatus(status)) return status;
	else return undefined;
}

//

export const TAB_QUERY_PARAM = 'tab';

export const TABS = {
	pipeline: m.Pipelines(),
	other: m.Conformance_and_custom_checks()
} as const;

export type Tab = keyof typeof TABS;

export function getCurrentTab(url: URL): Tab {
	const tab = url.searchParams.get(TAB_QUERY_PARAM);
	if (tab === null) {
		return 'pipeline';
	}
	if (!Object.keys(TABS).includes(tab as Tab)) {
		return 'pipeline';
	}
	return tab as Tab;
}

//

export async function fetchWorkflows(
	tab: Tab,
	options: {
		fetch?: typeof fetch;
		status?: ExtendedWorkflowStatus | null;
		limit?: number;
		offset?: number;
	} = {}
) {
	let status: ExtendedWorkflowStatus | undefined = undefined;
	if (options.status !== null) {
		status = options.status;
	}
	if (tab === 'pipeline') {
		return Pipeline.Workflows.listAll({
			fetch: options.fetch,
			status,
			limit: options.limit,
			offset: options.offset
		});
	} else {
		const res = await Workflow.fetchWorkflows({ fetch: options.fetch, status });
		if (res instanceof Error) {
			throw res;
		}
		return res;
	}
}

export type PaginationParams = {
	limit?: number;
	offset?: number;
};

export function getPaginationQueryParams(url: URL): PaginationParams {
	const limit = url.searchParams.get(Pipeline.Workflows.LIMIT_PARAM);
	const offset = url.searchParams.get(Pipeline.Workflows.OFFSET_PARAM);
	return {
		limit: parseLimit(limit),
		offset: parseOffset(offset)
	};
}

const DEFAULT_LIMIT = 20;
const DEFAULT_OFFSET = 0;

export function parseLimit(value: string | null): number | undefined {
	const parsed = Number(value);
	return Number.isNaN(parsed) || value === null ? DEFAULT_LIMIT : parsed;
}

export function parseOffset(value: string | null): number | undefined {
	const parsed = Number(value);
	return Number.isNaN(parsed) || value === null ? DEFAULT_OFFSET : parsed;
}
