// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getStandardsWithTestSuites, listSuites } from '$lib/conformance';
import { getUserOrganization } from '$lib/utils';

export const load = async ({ fetch }) => {
	const organization = await getUserOrganization({ fetch });
	// Loading organization for displaying ownership status

	const conformanceChecks = await getStandardsWithTestSuites({ fetch, surface: 'pipeline' });

	if (conformanceChecks instanceof Error) {
		error(500, { message: conformanceChecks.message });
	}

	const suitesResult = await listSuites({ fetch, surface: 'pipeline' });
	if (suitesResult.isErr) {
		error(500, { message: suitesResult.error.message });
	}

	return {
		organization,
		conformanceChecks,
		conformanceSuites: suitesResult.value
	};
};
