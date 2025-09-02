// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getStandardsWithTestSuites } from '$lib/standards';

import type { CustomChecksResponse } from '@/pocketbase/types/index.generated.js';

import { pb } from '@/pocketbase/index.js';

//

export const load = async ({ fetch, parent }) => {
	const { organization } = await parent();

	const result = await getStandardsWithTestSuites({ fetch });

	let customChecks: CustomChecksResponse[] = [];
	try {
		customChecks = await pb.collection('custom_checks').getFullList({
			filter: `owner = '${organization.id}' || public = true`,
			fetch
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
