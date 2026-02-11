// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getStatusQueryParam } from '$lib/workflows/index.js';

import { redirect } from '@/i18n/index.js';

import { fetchWorkflows, getCurrentTab } from './_partials';

//

export const load = async ({ fetch, url }) => {
	const status = getStatusQueryParam(url);
	const tab = getCurrentTab(url);

	if (status instanceof Error) {
		redirect('/my/tests/runs');
		throw 'err'; // ts escape hatch
	}

	const workflows = await fetchWorkflows(tab, { fetch, status: status });
	if (workflows instanceof Error) {
		error(500, {
			message: workflows.message
		});
	}

	return {
		workflows
	};
};
