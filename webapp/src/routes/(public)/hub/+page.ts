// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { listHubSuites } from '$lib/conformance';

/** Product suite rows for the hub table SSR seed (flat `conformance_suites`). */
export const load = async ({ fetch }) => {
	const suitesResult = await listHubSuites({ fetch, surface: 'pipeline' });
	if (suitesResult.isErr) {
		error(500, { message: suitesResult.error.message });
	}

	return {
		conformanceSuites: suitesResult.value
	};
};
