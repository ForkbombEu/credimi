// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

export const load = ({ url }) => {
	const deeplink = url.searchParams.get('deeplink');
	const workflowId = url.searchParams.get('workflow-id');
	if (!deeplink || !workflowId) error(404);

	return {
		deeplink,
		workflowId
	};
};
