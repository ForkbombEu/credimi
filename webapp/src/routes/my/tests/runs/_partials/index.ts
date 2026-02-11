// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pipeline, Workflow } from '$lib';

import { m } from '@/i18n';

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
		status?: Workflow.WorkflowStatus | null;
	} = {}
) {
	let status: Workflow.WorkflowStatus | undefined = undefined;
	if (options.status !== null) {
		status = options.status;
	}
	if (tab === 'pipeline') {
		return Pipeline.Workflows.listAll({ fetch: options.fetch, status });
	} else {
		const res = await Workflow.fetchWorkflows({ fetch: options.fetch, status });
		if (res instanceof Error) {
			throw res;
		}
		return res;
	}
}
