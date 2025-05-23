// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CustomChecksResponse } from '@/pocketbase/types/index.generated.js';
import { getStandardsAndTestSuites } from './_partials/standards-response-schema.js';
import { error } from '@sveltejs/kit';
import { Either } from 'effect';
import { pb } from '@/pocketbase/index.js';
import { checkAuthFlagAndUser } from '$lib/utils/index.js';

//

export const load = async ({ fetch, parent }) => {
	await checkAuthFlagAndUser({ fetch });
	
	const result = await getStandardsAndTestSuites({ fetch });
	const { organization } = await parent();
	if (!organization) throw error(403);


	let customChecks: CustomChecksResponse[] = [];
	try {
		customChecks = await pb.collection('custom_checks').getFullList({filter: `owner = '${organization.id}'`});
	} catch (e) {
		console.error(e);
	}

	if (Either.isLeft(result)) {
		error(500, { message: result.left.message });
	} else {
		return {
			standardsAndTestSuites: result.right,
			customChecks
		};
	}
};
