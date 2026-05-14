// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getStandardsWithTestSuites } from '$lib/standards';
import { getUserOrganization } from '$lib/utils';

export const load = async ({ fetch }) => {
	const organization = await getUserOrganization({ fetch });
	// Loading organization for displaying ownership status

	const conformanceChecks = await getStandardsWithTestSuites({ fetch, forPipeline: true });

	if (conformanceChecks instanceof Error) {
		error(500, { message: conformanceChecks.message });
	}

	return {
		organization,
		conformanceChecks
	};
};
