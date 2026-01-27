// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getRecordByCanonifiedPath } from '$lib/canonify';
import { CUSTOM_CHECK_QUERY_PARAM } from '$lib/marketplace';
import { getStandardsWithTestSuites } from '$lib/standards';

import type { CustomChecksResponse } from '@/pocketbase/types/index.generated.js';

import { pb } from '@/pocketbase/index.js';

//

export const load = async ({ fetch, parent, url }) => {
	const { organization } = await parent();

	const result = await getStandardsWithTestSuites({ fetch });

	let customChecks: CustomChecksResponse[] = [];
	try {
		customChecks = await pb.collection('custom_checks').getFullList({
			filter: `owner = '${organization.id}' || published = true`,
			fetch
		});
	} catch (e) {
		console.error(e);
	}

	let customCheckId: string | undefined;
	const customCheckPath = url.searchParams.get(CUSTOM_CHECK_QUERY_PARAM);
	if (customCheckPath) {
		const record = await getRecordByCanonifiedPath<CustomChecksResponse>(customCheckPath, {
			fetch
		});
		if (!(record instanceof Error)) customCheckId = record.id;
		else console.error(record);
	}

	if (result instanceof Error) {
		error(500, { message: result.message });
	} else {
		return {
			standardsAndTestSuites: result,
			customChecks,
			customCheckId
		};
	}
};
