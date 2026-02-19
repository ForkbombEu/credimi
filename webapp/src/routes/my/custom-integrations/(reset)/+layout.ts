// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { checkAuthFlagAndUser, getUserOrganization } from '$lib/utils/index.js';

//

export const load = async ({ fetch }) => {
	await checkAuthFlagAndUser({ fetch });
	const organization = await getUserOrganization({ fetch });

	if (!organization) {
		error(500, { message: 'USER_MISSING_ORGANIZATION' });
	} else {
		return {
			organization
		};
	}
};
