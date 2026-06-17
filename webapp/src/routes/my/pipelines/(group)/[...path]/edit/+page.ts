// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EnrichedPipeline } from '$lib/pipeline-form/functions.js';

import { getEnrichedPipeline } from '$lib/pipeline-form/functions';

//

function minimalPipeline(record: EnrichedPipeline['record']): EnrichedPipeline {
	return { record, steps: [], runtime: undefined };
}

export const load = async ({ fetch, parent }) => {
	const { pipeline: record } = await parent();

	if (record.manual) {
		return {
			pipeline: minimalPipeline(record),
			startLockedManual: true as const
		};
	}

	try {
		const enriched = await getEnrichedPipeline(record.id, { fetch });
		return { pipeline: enriched };
	} catch {
		return {
			pipeline: minimalPipeline(record),
			startLockedManual: true as const
		};
	}
};
