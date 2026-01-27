// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getRecordByCanonifiedPath } from '$lib/canonify';

import { pb } from '@/pocketbase';

import { getPipelineWorkflows } from '../_partials/workflows.js';

//

export const load = async ({ params, fetch }) => {
	// Try to get pipeline by canonified path first, fallback to ID
	let pipeline;
	const pathOrId = params.pipeline_id;
	
	// First, try to resolve as a canonified path
	// Canonified paths for pipelines have format: /organization-name/pipeline-name
	if (pathOrId.includes('/')) {
		const result = await getRecordByCanonifiedPath(pathOrId, { fetch });
		if (!(result instanceof Error)) {
			pipeline = result;
		}
	}
	
	// If not found by canonified path, try by ID
	// PocketBase IDs are 15 characters long with specific format
	if (!pipeline) {
		try {
			pipeline = await pb.collection('pipelines').getOne(pathOrId, { fetch });
		} catch (error) {
			// If both methods fail, let the error propagate
			throw error;
		}
	}
	
	const workflows = await getPipelineWorkflows(pipeline.id, { fetch });

	return { pipeline, workflows };
};
