// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { isWorkflowStatus, type WorkflowStatusType } from '$lib/temporal';
import { page } from '$app/state';
import { goto } from '@/i18n';

//

export const STATUS_PARAM = 'statuses';

export function getWorkflowStatusesFromUrl(url: URL): WorkflowStatusType[] {
	const statusList = url.searchParams.get(STATUS_PARAM);
	if (!statusList) return [];
	return statusList.split(',').filter(isWorkflowStatus);
}

export function setWorkflowStatusesInUrl(statuses: WorkflowStatusType[]) {
	const query = new URLSearchParams(page.url.searchParams.toString());
	if (statuses.length === 0) {
		query.delete(STATUS_PARAM);
	} else {
		query.set(STATUS_PARAM, statuses.join(','));
	}
	goto(`${page.url.pathname}?${query.toString()}`);
}

//

export const INVALIDATE_KEY = 'app:load-workflows';
