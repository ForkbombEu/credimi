// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EnrichedPipeline } from '$lib/pipeline-form/functions.js';
import type { PipelinesResponse } from '@/pocketbase/types/index.generated';

import { getEnrichedPipeline, minimalEnrichedPipeline } from '$lib/pipeline-form/functions';

//

export type EditPipelineLoad = {
	pipeline: EnrichedPipeline;
	startLockedManual?: true;
};

async function loadEditPipeline(
	record: PipelinesResponse,
	fetchFn: typeof fetch
): Promise<EditPipelineLoad> {
	if (!record.manual) {
		try {
			return { pipeline: await getEnrichedPipeline(record.id, { fetch: fetchFn }) };
		} catch {
			// Enrichment failed — fall back to locked manual edit of the raw YAML.
		}
	}

	return {
		pipeline: minimalEnrichedPipeline(record),
		startLockedManual: true
	};
}

/**
 * Return enrichment as an unresolved Promise so client-side navigation can
 * reach the edit page immediately; the page shows a loading overlay via `{#await}`.
 */
export const load = async ({ fetch, parent }) => {
	const { pipeline: record } = await parent();

	return {
		edit: loadEditPipeline(record, fetch)
	};
};
