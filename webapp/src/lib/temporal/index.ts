// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { workflowStatuses } from '@forkbombeu/temporal-ui';

import TemporalI18nProvider from './temporal-i18n-provider.svelte';
export { TemporalI18nProvider, workflowStatuses };

//

export type WorkflowStatusType = NonNullable<(typeof workflowStatuses)[number]>;

export function isWorkflowStatus(status?: string | null | undefined): status is WorkflowStatusType {
	return workflowStatuses.includes(status as WorkflowStatusType);
}

//

// Safelisting background colors for tailwind generated classes
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const statusColors = {
	Running: 'bg-blue-300',
	TimedOut: 'bg-orange-200',
	Completed: 'bg-green-200',
	Failed: 'bg-red-200',
	ContinuedAsNew: 'bg-purple-200',
	Canceled: 'bg-slate-100',
	Terminated: 'bg-yellow-200',
	Paused: 'bg-yellow-200',
	Unspecified: 'bg-slate-100',
	Scheduled: 'bg-blue-300',
	Started: 'bg-blue-300',
	Open: 'bg-green-200',
	New: 'bg-blue-300',
	Initiated: 'bg-blue-300',
	Fired: 'bg-pink-200',
	CancelRequested: 'bg-yellow-200',
	Signaled: 'bg-pink-200',
	Pending: 'bg-purple-200',
	Retrying: 'bg-red-200'
};
