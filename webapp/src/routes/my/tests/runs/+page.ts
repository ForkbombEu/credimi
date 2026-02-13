// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	fetchWorkflows,
	getCurrentTab,
	getPaginationQueryParams,
	getStatusQueryParam
} from './_partials';

//

export const load = async ({ fetch, url }) => {
	const status = getStatusQueryParam(url);
	const tab = getCurrentTab(url);
	const pagination = getPaginationQueryParams(url);

	const workflows = await fetchWorkflows(tab, { fetch, status, ...pagination });
	return {
		workflows
	};
};
