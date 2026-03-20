// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { fetchWorkflows, getPaginationQueryParams, getStatusQueryParam } from './_partials';

//

export const load = async ({ fetch, url }) => {
	const status = getStatusQueryParam(url);
	const pagination = getPaginationQueryParams(url);

	const workflows = await fetchWorkflows('other', { fetch, status, ...pagination });
	return {
		workflows,
		pagination
	};
};
