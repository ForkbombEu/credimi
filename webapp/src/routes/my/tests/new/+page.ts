// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CustomChecksResponse } from '@/pocketbase/types/index.generated.js';
import { getStandardsWithTestSuites } from '$lib/standards';
import { error } from '@sveltejs/kit';
import { pb } from '@/pocketbase/index.js';
import { checkAuthFlagAndUser } from '$lib/utils/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';

//

export const load = async ({ fetch }) => {
	await checkAuthFlagAndUser({ fetch });

	const result = await getStandardsWithTestSuites({ fetch });
	const organizationAuth = await new PocketbaseQueryAgent(
		{
			collection: 'orgAuthorizations',
			expand: ['organization'],
			filter: `user.id = "${pb.authStore.record?.id}"`
		},
		{ fetch }
	).getFullList();

	const organization = organizationAuth.at(0)?.expand?.organization;
	if (!organization) error(500, { message: 'USER_MISSING_ORGANIZATION' });

	let customChecks: CustomChecksResponse[] = [];
	try {
		customChecks = await pb.collection('custom_checks').getFullList({
			filter: `owner = '${organization.id}' || public = true`
		});
	} catch (e) {
		console.error(e);
	}

	if (result instanceof Error) {
		error(500, { message: result.message });
	} else {
		return {
			standardsAndTestSuites: result,
			customChecks
		};
	}
};
