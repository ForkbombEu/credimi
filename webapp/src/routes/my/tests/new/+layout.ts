// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { pb } from '@/pocketbase/index.js';
import { checkAuthFlagAndUser } from '$lib/utils/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';

//

export const load = async ({ fetch }) => {
	await checkAuthFlagAndUser({ fetch });

	const organizationAuth = await new PocketbaseQueryAgent(
		{
			collection: 'orgAuthorizations',
			expand: ['organization'],
			filter: `user.id = "${pb.authStore.record?.id}"`
		},
		{ fetch }
	).getFullList();

	const organization = organizationAuth.at(0)?.expand?.organization;

	if (!organization) {
		error(500, { message: 'USER_MISSING_ORGANIZATION' });
	} else {
		return {
			organization
		};
	}
};
